package clusterclaim

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterClaimReconciler(
	log logr.Logger,
	clusterName string,
	clusterClient clusterclientset.Interface,
	hubClient client.Client,
	clusterClaimLister clusterv1alpha1lister.ClusterClaimLister,
	nodeLister corev1lister.NodeLister,
	generateExpectClusterClaimsByClaimer func() ([]*clusterv1alpha1.ClusterClaim, error),
	enableSyncLabelsToClaim bool) (*clusterClaimReconciler, error) {

	hubManaged, err := labels.NewRequirement(labelHubManaged, selection.Exists, nil)
	if err != nil {
		return nil, err
	}
	notCustomizedOnly, err := labels.NewRequirement(labelCustomizedOnly, selection.DoesNotExist, nil)
	if err != nil {
		return nil, err
	}
	customizedOnly, err := labels.NewRequirement(labelCustomizedOnly, selection.Exists, nil)
	if err != nil {
		return nil, err
	}

	return &clusterClaimReconciler{
		log:                log,
		clusterName:        clusterName,
		clusterClient:      clusterClient,
		clusterClaimLister: clusterClaimLister,
		hubClient:          hubClient,
		generateExpectClusterClaims: func() ([]*clusterv1alpha1.ClusterClaim, error) {
			expectedClaims, err := generateExpectClusterClaimsByClaimer()
			if err != nil {
				return nil, err
			}

			// append cluster schedulable claim
			isSchedulable, err := getClusterSchedulable(nodeLister)
			if err != nil {
				return nil, err
			}

			// The reason why not using the standard "newClusterClaim" to creat the claim is that
			// in the case of SD multi-hub, we have 2 agents in different versions working on a same managed cluster.
			// The schedulabe claim will be created by on agent and removed by another agent.
			// TODO: The hub-managed label should be removed, the `cleanClusterClaims` for all controller created claims should be removed.
			expectedClaims = append(expectedClaims, &clusterv1alpha1.ClusterClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ClaimClusterSchedulable,
					Labels: map[string]string{labelExcludeBackup: "true"},
				},
				Spec: clusterv1alpha1.ClusterClaimSpec{
					Value: strconv.FormatBool(isSchedulable),
				},
			})

			return expectedClaims, nil
		},
		hubManagedSelector:      labels.NewSelector().Add(*hubManaged).Add(*notCustomizedOnly),
		customizedOnlyselector:  labels.NewSelector().Add(*hubManaged).Add(*customizedOnly),
		enableSyncLabelsToClaim: enableSyncLabelsToClaim,
	}, nil
}

// clusterClaimReconciler reconciles cluster claim objects
type clusterClaimReconciler struct {
	log                         logr.Logger
	clusterName                 string
	clusterClient               clusterclientset.Interface
	clusterClaimLister          clusterv1alpha1lister.ClusterClaimLister
	hubClient                   client.Client
	generateExpectClusterClaims func() ([]*clusterv1alpha1.ClusterClaim, error)

	hubManagedSelector     labels.Selector // used to filter claims that generaed by the control-plane
	customizedOnlyselector labels.Selector // used to filter claims that created by user via managedclusterinfo labels

	enableSyncLabelsToClaim bool
}

func (r *clusterClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.V(5).Info("Sync cluster claims")
	// Sync claims that created by control-plane
	err := r.syncControlPlaneCreatedClaims(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Sync claims that created by user via managedclusterinfo labels
	if r.enableSyncLabelsToClaim {
		return ctrl.Result{}, r.syncLabelsToClaims(ctx)
	}
	return ctrl.Result{}, err
}

func (r *clusterClaimReconciler) syncControlPlaneCreatedClaims(ctx context.Context) error {
	return syncClaims(ctx, r.clusterClient,
		r.generateExpectClusterClaims,
		updateChecks,
		func() ([]*clusterv1alpha1.ClusterClaim, error) {
			return r.clusterClaimLister.List(r.hubManagedSelector)
		})
}

func (r *clusterClaimReconciler) syncLabelsToClaims(ctx context.Context) error {
	return syncClaims(ctx, r.clusterClient,
		func() ([]*clusterv1alpha1.ClusterClaim, error) {
			return genLabelsToClaims(r.hubClient, r.clusterName)
		},
		nil, // no need to check update for user created claims
		func() ([]*clusterv1alpha1.ClusterClaim, error) {
			return r.clusterClaimLister.List(r.customizedOnlyselector)
		})
}

// syncClaims contains 3 main steps:
// 1. Get expected claims
// 2. Create/Update claims - in this step, we use updatechecks to filter out claims that we don't want to update
// 3. Get existing claims, compare with expected claims, and clean unexpected ones.
func syncClaims(ctx context.Context,
	clusterClient clusterclientset.Interface,
	getExpectedClaims func() ([]*clusterv1alpha1.ClusterClaim, error),
	updatechecks []updateCheck,
	getExistingClaims func() ([]*clusterv1alpha1.ClusterClaim, error)) error {
	// Get expected claims
	expectClaims, err := getExpectedClaims()
	if err != nil {
		return err
	}

	// Create/Update claims.
	errs := []error{}
	for _, c := range expectClaims {
		if err := createOrUpdateClusterClaim(ctx, clusterClient, c, updatechecks); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return utils.NewMultiLineAggregate(errs)
	}

	// List existing claims then fileter out stable claims
	existClusterClaims, err := getExistingClaims()
	if err != nil {
		return err
	}

	//  Clean unexpected ones.
	return cleanClusterClaims(ctx, clusterClient, existClusterClaims, expectClaims)
}

func genLabelsToClaims(hubClient client.Client, clusterName string) ([]*clusterv1alpha1.ClusterClaim, error) {
	var claims []*clusterv1alpha1.ClusterClaim
	request := types.NamespacedName{Namespace: clusterName, Name: clusterName}
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := hubClient.Get(context.TODO(), request, clusterInfo)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return claims, nil
		}
		return claims, err
	}

	// do not create claim if the label is managed by ACM.
	// if the label format is aaa/bbb, the name of claim will be bbb.aaa.
	// Besides, "_" and "/" in the label name will be replaced with "-" and "." respectively.
	for label, value := range clusterInfo.Labels {
		if internalLabels.Has(label) || strings.Contains(label, "open-cluster-management.io") {
			continue
		}

		// convert the string to lower case
		name := strings.ToLower(label)

		// and then replace invalid characters
		subs := strings.Split(name, "/")
		if len(subs) == 2 {
			name = fmt.Sprintf("%s.%s", subs[1], subs[0])
		} else if len(subs) > 2 {
			name = strings.ReplaceAll(name, "/", ".")
		}
		name = strings.ReplaceAll(name, "_", "-")

		// ignore the label if the transformed name is still not a valid resource name
		if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
			klog.V(4).Infof("skip syncing label %q of ManagedClusterInfo to ClusterCliam because it's an invalid resource name", label)
			continue
		}

		// ignore the label if its value is empty. (the value of ClusterCliam can not be empty)
		if len(value) == 0 {
			klog.V(4).Infof("skip syncing label %q of ManagedClusterInfo to ClusterClaim because its value is empty.", label)
			continue
		}

		claim := newClusterClaim(name, value)
		if claim.Labels != nil {
			claim.Labels[labelCustomizedOnly] = ""
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

func filterOutStableClaims(claims []*clusterv1alpha1.ClusterClaim) (filtered []*clusterv1alpha1.ClusterClaim) {
	for _, c := range claims {
		if !stableClaims.Has(c.Name) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

type updateCheck func(oldClaim, newClaim *clusterv1alpha1.ClusterClaim) bool

// There are 3 cases that we don't want to update ClusterClaim:
//  1. newClaim.Name is in clusterClaimCreateOnlyList
//  2. oldClaim.Name is ClaimOCMProduct and oldClaim.Spec.Value is not empty
//     and newClaim.Spec.Value is ProductOther
//  3. oldClaim.Name is ClaimOCMPlatform and oldClaim.Spec.Value is not empty
//     and newClaim.Spec.Value is PlatformOther
//
// In a word, for product and platform, we don't want update them from a specific value to "Other".
// This is because in the process of upgrading, platform and product could be detected as "Other"
// then reflect on the console.
var updateChecks = []updateCheck{
	func(oldClaim, newClaim *clusterv1alpha1.ClusterClaim) bool {
		return !utils.ContainsString(clusterClaimCreateOnlyList, newClaim.Name) &&
			oldClaim.Spec.Value != ""
	},
	func(oldClaim, newClaim *clusterv1alpha1.ClusterClaim) bool {
		// only check ClaimOCMProduct
		if oldClaim.Name != ClaimOCMProduct {
			return true
		}
		// don't allow update from a specific value to "Other"
		if newClaim.Spec.Value == ProductOther {
			return false
		}
		return true
	},
	func(oldClaim, newClaim *clusterv1alpha1.ClusterClaim) bool {
		// only check ClaimOCMProduct
		if oldClaim.Name != ClaimOCMPlatform {
			return true
		}
		// don't allow update from a specific value to "Other"
		if newClaim.Spec.Value == PlatformOther {
			return false
		}
		return true
	},
}

func createOrUpdateClusterClaim(ctx context.Context, clusterClient clusterclientset.Interface,
	newClaim *clusterv1alpha1.ClusterClaim,
	updateChecks []updateCheck) error {
	oldClaim, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(ctx, newClaim.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Create(ctx, newClaim, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("unable to create ClusterClaim: %v, %w", newClaim, err)
		}
	case err != nil:
		return fmt.Errorf("unable to get ClusterClaim %q: %w", newClaim.Name, err)
	case !reflect.DeepEqual(oldClaim.Spec, newClaim.Spec):
		if len(updateChecks) != 0 {
			for _, check := range updateChecks {
				if !check(oldClaim, newClaim) {
					// not pass the check, skip update
					return nil
				}
			}
		}
		oldClaim.Spec = newClaim.Spec
		_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Update(ctx, oldClaim, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update ClusterClaim %q: %w", oldClaim.Name, err)
		}
	}
	return nil
}

func cleanClusterClaims(ctx context.Context, clusterClient clusterclientset.Interface,
	currentClusterClaims, expectClusterClaims []*clusterv1alpha1.ClusterClaim) error {
	errs := []error{}
	expectSet := sets.Set[string]{}
	for _, c := range expectClusterClaims {
		expectSet.Insert(c.Name)
	}
	for _, c := range currentClusterClaims {
		if expectSet.Has(c.Name) {
			continue
		}
		err := clusterClient.ClusterV1alpha1().ClusterClaims().Delete(ctx, c.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return utils.NewMultiLineAggregate(errs)
}

func getClusterSchedulable(nodeLister corev1lister.NodeLister) (bool, error) {
	// list all nodes:
	// * if any of nodes is "schedulable", update the clusterclaim value to 'true'.
	// * otherwise, update the clusterclaim value to 'false'.
	nodes, err := nodeLister.List(labels.Everything())
	if err != nil {
		return false, err
	}

	// Check if any node is schedulable
	isSchedulable := false
	for _, node := range nodes {
		if IsNodeSchedulable(node) {
			isSchedulable = true
			break
		}
	}

	return isSchedulable, nil
}
