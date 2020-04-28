/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	viewv1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1"
)

// SpokeViewReconciler reconciles a SpokeView object
type SpokeViewReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	SpokeDynamicClient dynamic.Interface
}

func (r *SpokeViewReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("spokeview", req.NamespacedName)

	return ctrl.Result{}, nil
}

func (r *SpokeViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&viewv1.SpokeView{}).
		Complete(r)
}
