// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"reflect"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	helmutil "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/helm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	helmrelease "k8s.io/helm/pkg/proto/hapi/release"
)

func (k *Klusterlet) handleResourceWork(work *v1alpha1.Work) error {
	result, err := k.queryResource(work.Spec.Scope)
	// Post work result
	if err == nil && result != nil {
		data, decerr := json.Marshal(result)
		if decerr != nil {
			return decerr
		}

		labels := work.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		labels[mcm.ClusterLabel] = work.Spec.Cluster.Name

		// Compress data
		// TODO should add a flag in work to enable/disable compression
		var compressed bytes.Buffer
		w, decerr := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
		if decerr != nil {
			return decerr
		}
		_, decerr = w.Write(data)
		w.Close()
		if decerr != nil {
			return decerr
		}

		rvResult := &v1alpha1.ResourceViewResult{
			ObjectMeta: metav1.ObjectMeta{
				Name:      work.Name,
				Namespace: work.Namespace,
				Labels:    labels,
			},
			Data: compressed.Bytes(),
		}

		restClient := k.hcmclientset.McmV1alpha1().RESTClient()
		if reflect.ValueOf(restClient).IsNil() {
			return nil
		}

		_, err = restClient.
			Post().
			Namespace(work.Namespace).
			Resource("works").
			Name(work.Name).
			SubResource("result").
			Body(rvResult).
			Do().Get()
		if err != nil {
			return err
		}
	}

	// Update work status
	if err != nil {
		return k.updateFailedStatus(work, err)
	}

	if work.Spec.Scope.Mode == v1alpha1.PeriodicResourceUpdate {
		work.Status.Type = v1alpha1.WorkProcessing
	} else {
		work.Status.Type = v1alpha1.WorkCompleted
	}

	work.Status.LastUpdateTime = metav1.Now()
	// Clean up work reason
	work.Status.Reason = ""
	_, err = k.hcmclientset.McmV1alpha1().Works(k.config.ClusterNamespace).UpdateStatus(work)
	if err != nil {
		return err
	}

	return nil
}

func (k *Klusterlet) queryResource(scope v1alpha1.ResourceFilter) (runtime.Object, error) {
	labelSelector, err := utils.ConvertLabels(scope.LabelSelector)
	if err != nil {
		return nil, err
	}

	fieldSelector := scope.FieldSelector

	if scope.ResourceType == v1alpha1.ResourceReleases {
		releaselists := []v1alpha1.HelmRelease{}
		if k.helmControl != nil {
			statusCode := []helmrelease.Status_Code{
				helmrelease.Status_UNKNOWN,
				helmrelease.Status_DEPLOYED,
				helmrelease.Status_DELETED,
				helmrelease.Status_DELETING,
				helmrelease.Status_FAILED,
				helmrelease.Status_PENDING_INSTALL,
				helmrelease.Status_PENDING_UPGRADE,
				helmrelease.Status_PENDING_ROLLBACK,
			}
			releases, e := k.helmControl.GetHelmReleases(
				scope.Name, statusCode, scope.NameSpace, 256)
			if e != nil {
				return nil, e
			}

			if releases != nil {
				for _, release := range releases.Releases {
					rl := helmutil.ConvertHelmReleaseFromRelease(release)
					releaselists = append(releaselists, rl)
				}
			}
		}

		if scope.ServerPrint {
			releaseTable, relerr := helmutil.PrintReleaseTable(&v1alpha1.ResultHelmList{Items: releaselists})
			if relerr != nil {
				return nil, relerr
			}
			return releaseTable, nil
		}

		return &v1alpha1.ResultHelmList{Items: releaselists}, nil
	}

	var obj runtime.Object
	if scope.Name == "" {
		options := &metav1.ListOptions{
			LabelSelector: labelSelector.String(),
			FieldSelector: fieldSelector,
			// limit the number of resource to list to be 1000 to be safely stored in etcd
			Limit: 1000,
		}
		obj, err = k.kubeControl.List(scope.ResourceType, scope.NameSpace, options, scope.ServerPrint)
		if err != nil {
			return nil, err
		}
	} else {
		obj, err = k.kubeControl.Get(nil, scope.ResourceType, scope.NameSpace, scope.Name, scope.ServerPrint)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}
