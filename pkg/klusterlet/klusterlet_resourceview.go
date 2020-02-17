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
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *Klusterlet) handleResourceWork(work *v1beta1.Work) error {
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

		rvResult := &v1beta1.ResourceViewResult{
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

	if work.Spec.Scope.Mode == v1beta1.PeriodicResourceUpdate {
		work.Status.Type = v1beta1.WorkProcessing
	} else {
		work.Status.Type = v1beta1.WorkCompleted
	}

	work.Status.LastUpdateTime = metav1.Now()
	// Clean up work reason
	work.Status.Reason = ""
	_, err = k.hcmclientset.McmV1beta1().Works(k.config.ClusterNamespace).UpdateStatus(work)
	if err != nil {
		return err
	}

	return nil
}

func (k *Klusterlet) queryResource(scope v1beta1.ResourceFilter) (runtime.Object, error) {
	labelSelector, err := utils.ConvertLabels(scope.LabelSelector)
	if err != nil {
		return nil, err
	}

	fieldSelector := scope.FieldSelector

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
