package microservice

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileMicroService) reconcileInstance(microService *appv1.MicroService) error {

	for _, version := range microService.Spec.Versions {

		deploy, err := makeVersionDeployment(&version, microService)
		if err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(microService, deploy, r.scheme); err != nil {
			return err
		}

		// Check if the Deployment already exists
		found := &appsv1.Deployment{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Create(context.TODO(), deploy)
			return err
		} else if err != nil {
			return err
		}

		// Update the found object and write the result back if there are any changes
		if !reflect.DeepEqual(deploy.Spec, found.Spec) {
			found.Spec = deploy.Spec
			log.Info("Updating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func makeVersionDeployment(version *appv1.DeployVersion, microService *appv1.MicroService) (*appsv1.Deployment, error) {

	labels := microService.Labels
	labels["app.o0w0o.cn/service"] = microService.Name
	labels["app.o0w0o.cn/version"] = version.Name

	deploySepc := version.Template

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      microService.Name + "-" + version.Name,
			Namespace: microService.Namespace,
			Labels:    labels,
		},
		Spec: deploySepc,
	}

	return deploy, nil
}
