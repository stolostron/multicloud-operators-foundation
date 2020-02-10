// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mattbaird/jsonpatch"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/mcm-webhook/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/plugin/pkg/useridentity"
	"k8s.io/api/admission/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

type admissionHandler struct {
	lister rbaclisters.RoleBindingLister
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

// admitFunc is the type we use for all of our validators and mutators
type admitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

// serve handles the http portion of a request prior to handing to an admit
// function
func (a *admissionHandler) serve(w io.Writer, r *http.Request, admit admitFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.V(2).Info(fmt.Sprintf("handling request: %s", body))

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1beta1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1beta1.AdmissionReview{}

	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Error(err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	klog.V(2).Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Response))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func (a *admissionHandler) mutateResource(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	klog.Info("mutating custom resource")
	raw := ar.Request.Object.Raw
	crd := apiextensionsv1beta1.CustomResourceDefinition{}
	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &crd); err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}
	ori, err := json.Marshal(crd)
	if err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}
	annotations := crd.GetAnnotations()

	resAnnotations := useridentity.MergeUserIdentityToAnnotations(ar.Request.UserInfo, annotations, crd.GetNamespace(), a.lister)
	crd.SetAnnotations(resAnnotations)
	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	crBytes, err := json.Marshal(crd)
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	res, err := jsonpatch.CreatePatch(ori, crBytes)
	if err != nil {
		klog.Errorf("Create patch error: %+v", err)
		return nil
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	reviewResponse.Patch = resBytes
	pt := v1beta1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt
	klog.Infof("Successfully Added user and group for resource: %+v, name: %+v", ar.Request.Resource.Resource, crd.GetName())
	return &reviewResponse
}

func (a *admissionHandler) serveMutateResource(w http.ResponseWriter, r *http.Request) {
	a.serve(w, r, a.mutateResource)
}

func main() {
	var config options.Config
	config.AddFlags()
	logs.InitLogs()
	defer logs.FlushLogs()
	flag.Parse()
	klog.Info("starting mcm webhook server")

	// build kube client
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigFile)
	if err != nil {
		klog.Fatalf("Error building kube clientset: %s", err.Error())
	}

	kubeclientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	informerFactory := informers.NewSharedInformerFactory(kubeclientset, 10*time.Minute)
	informer := informerFactory.Rbac().V1().RoleBindings()

	ah := &admissionHandler{
		lister: informer.Lister(),
	}

	stopCh := wait.NeverStop

	go informerFactory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, informer.Informer().HasSynced); !ok {
		klog.Fatalf("failed to wait for kubernetes caches to sync")
	}

	http.HandleFunc("/", ah.serveMutateResource)
	server := &http.Server{
		Addr:      ":8000",
		TLSConfig: options.ConfigTLS(config),
	}
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		klog.Errorf("Listen server tls error: %+v", err)
		return
	}
}
