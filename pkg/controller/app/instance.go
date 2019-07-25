package app

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileApp) reconcileMicroService(app *appv1.App) error {
	// Define the desired MicroService object
	for _, microService := range app.Spec.MicroServices {

		ms := &appv1.MicroService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name + microService.Name,
				Namespace: app.Namespace,
			},
			Spec: microService.Spec,
		}

		if err := controllerutil.SetControllerReference(app, ms, r.scheme); err != nil {
			return err
		}

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

		// Update the found object and write the result back if there are any changes
		if !reflect.DeepEqual(ms.Spec, found.Spec) {
			found.Spec = ms.Spec
			log.Info("Updating MicroService", "namespace", ms.Namespace, "name", ms.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
