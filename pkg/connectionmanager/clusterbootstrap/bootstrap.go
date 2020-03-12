// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterbootstrap

import (
	"context"
	"crypto/sha512"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"time"

	hcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	clientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
)

// BootStrapper is cluster bootstraper
type BootStrapper struct {
	host            string
	bootstrapclient clientset.Interface

	clusterNamespace string
	clusterName      string

	clientCert []byte
	clientKey  []byte
}

var clientCertUsage = []certificates.KeyUsage{
	certificates.UsageDigitalSignature,
	certificates.UsageKeyEncipherment,
	certificates.UsageClientAuth,
}

// NewBootStrapper create a bootstrap for cluster registration
func NewBootStrapper(
	bootstrapclient clientset.Interface,
	host, clusterNamespace, clusterName string, clientKey, clientCert []byte) *BootStrapper {
	bootstrapper := &BootStrapper{
		bootstrapclient:  bootstrapclient,
		host:             host,
		clusterName:      clusterName,
		clusterNamespace: clusterNamespace,
		clientCert:       clientCert,
		clientKey:        clientKey,
	}

	return bootstrapper
}

// LoadClientCert load config
func (bt *BootStrapper) LoadClientCert() ([]byte, []byte, []byte, error) {
	if bt.clientKey == nil {
		key, err := keyutil.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return nil, nil, nil, err
		}
		bt.clientKey = key
	}

	// if cert does not exist, start the informer to check hcmjoinrequest
	if bt.clientCert == nil {
		cert, err := bt.requestCertificate(context.Background(), bt.bootstrapclient, bt.clientKey, false)
		if err != nil {
			return nil, nil, nil, err
		}

		bt.clientCert = cert
	}

	configData := common.NewClientConfig(bt.host, bt.clientCert, bt.clientKey)
	return bt.clientKey, bt.clientCert, configData, nil
}

// RenewClientCert renews client certification
func (bt *BootStrapper) RenewClientCert(ctx context.Context) ([]byte, []byte, []byte, error) {
	if bt.clientKey == nil {
		key, err := keyutil.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return nil, nil, nil, err
		}
		bt.clientKey = key
	}

	// create a cluster join request with renewal annotation and wait for approval
	cert, err := bt.requestCertificate(ctx, bt.bootstrapclient, bt.clientKey, true)
	if err != nil {
		return nil, nil, nil, err
	}

	bt.clientCert = cert

	configData := common.NewClientConfig(bt.host, bt.clientCert, bt.clientKey)
	return bt.clientKey, bt.clientCert, configData, nil
}

func (bt *BootStrapper) requestCertificate(ctx context.Context, client clientset.Interface, privateKeyData []byte, renewal bool) (certData []byte, err error) {
	subject := Subject(bt.clusterName, bt.clusterNamespace)
	privateKey, err := keyutil.ParsePrivateKeyPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("invalid private key for certificate request: %v", err)
	}
	csrData, err := certutil.MakeCSR(privateKey, subject, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to generate certificate request: %v", err)
	}

	hcmjoinName := DigestedName(privateKeyData, subject)
	hcmjoin := &hcmv1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: hcmjoinName,
		},
		Spec: hcmv1alpha1.ClusterJoinRequestSpec{
			ClusterName:      bt.clusterName,
			ClusterNamespace: bt.clusterNamespace,
			Request:          csrData,
		},
		Status: hcmv1alpha1.ClusterJoinRequestStatus{
			Phase: hcmv1alpha1.JoinPhasePending,
		},
	}

	if renewal {
		hcmjoin.Annotations = map[string]string{
			common.RenewalAnnotation: "true",
		}
	}

	_, err = client.McmV1alpha1().ClusterJoinRequests().Create(hcmjoin)
	switch {
	case err == nil:
	case errors.IsAlreadyExists(err) && len(hcmjoinName) > 0:
		klog.Infof("hcm join request for this cluster already exists, reusing")
		_, err = client.McmV1alpha1().ClusterJoinRequests().Get(hcmjoinName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	return bt.waitForCertificate(ctx, client, hcmjoinName, 3600*time.Second)
}

func (bt *BootStrapper) waitForCertificate(ctx context.Context, client clientset.Interface, name string, timeout time.Duration) (certData []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fieldSelector := fields.OneTermEqualSelector("metadata.name", name).String()

	event, err := watchtools.UntilWithSync(
		ctx,
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return client.McmV1alpha1().ClusterJoinRequests().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return client.McmV1alpha1().ClusterJoinRequests().Watch(options)
			},
		},
		&hcmv1alpha1.ClusterJoinRequest{},
		nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Modified, watch.Added:
			case watch.Deleted:
				return false, fmt.Errorf("hcm join request %q was deleted", name)
			default:
				return false, nil
			}
			hcmjoin := event.Object.(*hcmv1alpha1.ClusterJoinRequest)
			if hcmjoin.Status.Phase == hcmv1alpha1.JoinPhaseDenied {
				return false, fmt.Errorf("hcm join request is not approved")
			}
			if hcmjoin.Status.Phase == hcmv1alpha1.JoinPhaseApproved && len(hcmjoin.Status.Certificate) != 0 {
				return true, nil
			}
			return false, nil
		},
	)

	if err == wait.ErrWaitTimeout {
		return nil, wait.ErrWaitTimeout
	}

	if err != nil {
		return nil, err
	}

	return event.Object.(*hcmv1alpha1.ClusterJoinRequest).Status.Certificate, nil
}

// This digest should include all the relevant pieces of the CSR we care about.
// We can't direcly hash the serialized CSR because of random padding that we
// regenerate every loop and we include usages which are not contained in the
// CSR. This needs to be kept up to date as we add new fields to the node
// certificates and with ensureCompatible.
func DigestedName(privateKeyData []byte, subject *pkix.Name) string {
	hash := sha512.New512_256()

	// Here we make sure two different inputs can't write the same stream
	// to the hash. This delimiter is not in the base64.URLEncoding
	// alphabet so there is no way to have spill over collisions. Without
	// it 'CN:foo,ORG:bar' hashes to the same value as 'CN:foob,ORG:ar'
	const delimiter = '|'
	encode := base64.RawURLEncoding.EncodeToString

	write := func(data []byte) {
		// there must be no errors
		_, _ = hash.Write([]byte(encode(data)))
		_, _ = hash.Write([]byte{delimiter})
	}

	write(privateKeyData)
	write([]byte(subject.CommonName))
	for _, v := range subject.Organization {
		write([]byte(v))
	}
	for _, v := range clientCertUsage {
		write([]byte(v))
	}

	return "clusterjoin-" + encode(hash.Sum(nil))
}

// Subject returns the pkix subject
func Subject(clusterName, clusterNamespace string) *pkix.Name {
	return &pkix.Name{
		Organization: []string{"hcm:clusters"},
		CommonName:   "hcm:clusters:" + clusterNamespace + ":" + clusterName,
	}
}
