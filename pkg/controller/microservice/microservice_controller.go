/*
Copyright 2019 Hypo.

Licensed under the GNU General Public License, Version 3 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/Coderhypo/KubeService/blob/master/LICENSE

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package microservice

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/types"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MicroService Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMicroService{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("microservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MicroService
	err = c.Watch(&source.Kind{Type: &appv1.MicroService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch resource created by MicroService
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &extensionsv1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMicroService{}

// ReconcileMicroService reconciles a MicroService object
type ReconcileMicroService struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a MicroService object and makes changes based on the state read
// and what is in the MicroService.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=microservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=microservices/status,verbs=get;update;patch
func (r *ReconcileMicroService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the MicroService instance
	instance := &appv1.MicroService{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		log.Info("Get deleted MicroService, and do nothing.")
		return reconcile.Result{}, nil
	}

	if err := r.reconcileInstance(instance); err != nil {
		log.Info("Reconcile Instance Versions error", err)
		return reconcile.Result{}, err
	}

	if err := r.reconcileLoadBalance(instance); err != nil {
		log.Info("Reconcile LoadBalance error", err)
		return reconcile.Result{}, err
	}

	oldMS := &appv1.MicroService{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, oldMS); err != nil {
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(oldMS.Spec, instance.Spec) {
		oldMS.Spec = instance.Spec
		if err := r.Update(context.TODO(), oldMS); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
