package app

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileApp) reconcileMicroService(req reconcile.Request, app *appv1.App) error {
	// Define the desired MicroService object
	labels := app.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.o0w0o.cn/app"] = app.Name
	newMicroServices := make(map[string]*appv1.MicroService)

	for i := range app.Spec.MicroServices {
		microService := &app.Spec.MicroServices[i]

		ms := &appv1.MicroService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name + "-" + microService.Name,
				Namespace: app.Namespace,
				Labels:    labels,
			},
			Spec: microService.Spec,
		}
		if err := controllerutil.SetControllerReference(app, ms, r.scheme); err != nil {
			return err
		}

		newMicroServices[ms.Name] = ms
		// Check if the MicroService already exists
		found := &appv1.MicroService{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}, found)

		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating MicroService", "namespace", ms.Namespace, "name", ms.Name)
			err = r.Create(context.TODO(), ms)
			return err

		} else if err != nil {

			return err

		}

		if !reflect.DeepEqual(ms.Spec, found.Spec) {

			found.Spec = ms.Spec
			log.Info("find MS changed and Updating MicroService", "namespace", ms.Namespace, "name", ms.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}

			err := r.Get(context.TODO(), types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}, found)
			if err != nil {
				return err
			}
			microService.Spec = found.Spec

		}
	}
	return r.cleanUpMicroServices(app, newMicroServices)
}

func (r *ReconcileApp) cleanUpMicroServices(app *appv1.App, msList map[string]*appv1.MicroService) error {
	// Check if the MicroService not exists
	ctx := context.Background()

	microServiceList := appv1.MicroServiceList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/app"] = app.Name

	if err := r.List(ctx, client.InNamespace(app.Namespace).
		MatchingLabels(labels), &microServiceList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return err
	}

	for i := range microServiceList.Items {
		oldMs := &microServiceList.Items[i]
		if _, exist := msList[oldMs.Name]; exist == false {
			log.Info("Deleted orphan MS and will delete it", "namespace", app.Namespace, "App", app.Namespace, "MS", oldMs.Name)
			err := r.Delete(context.TODO(), oldMs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
