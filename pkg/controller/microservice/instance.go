package microservice

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileMicroService) reconcileInstance(req reconcile.Request, microService *appv1.MicroService) error {

	newDeploys := make(map[string]*appsv1.Deployment)
	for _, version := range microService.Spec.Versions {

		deploy, err := makeVersionDeployment(&version, microService)
		if err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(microService, deploy, r.scheme); err != nil {
			return err
		}

		newDeploys[deploy.Name] = deploy
		// Check if the Deployment already exists
		found := &appsv1.Deployment{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Create(context.TODO(), deploy)
			return err
		} else if err != nil {
			return err
		} else if !reflect.DeepEqual(deploy.Spec, found.Spec) {
			// Update the found object and write the result back if there are any changes
			found.Spec = deploy.Spec
			log.Info("Updating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}
	}
	return r.cleanUpDeploy(req, microService, newDeploys)
}

func makeVersionDeployment(version *appv1.DeployVersion, microService *appv1.MicroService) (*appsv1.Deployment, error) {

	labels := microService.Labels
	if labels==nil{
		labels = make(map[string]string)
	}
	labels["app.o0w0o.cn/service"] = microService.Name
	labels["app.o0w0o.cn/version"] = version.Name

	deploySpec := version.Template

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      microService.Name + "-" + version.Name,
			Namespace: microService.Namespace,
			Labels:    labels,
		},
		Spec: deploySpec,
	}

	return deploy, nil
}

func (r *ReconcileMicroService) cleanUpDeploy(req reconcile.Request, microService *appv1.MicroService, newDeployList map[string]*appsv1.Deployment) error {
	// Check if the MicroService not exists
	ctx := context.Background()

	deployList := appsv1.DeploymentList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/service"] = microService.Name

	if err := r.List(ctx, client.InNamespace(req.Namespace).
		MatchingLabels(labels), &deployList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return err
	}

	for _, oldDeploy := range deployList.Items {
		if _, exist := newDeployList[oldDeploy.Name]; exist == false {
			err := r.Delete(context.TODO(), &oldDeploy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
