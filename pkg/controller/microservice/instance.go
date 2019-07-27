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
)

func (r *ReconcileMicroService) reconcileInstance(microService *appv1.MicroService) error {

	newDeploys := make(map[string]*appsv1.Deployment)
	for i := range microService.Spec.Versions {
		version := &microService.Spec.Versions[i]

		deploy, err := makeVersionDeployment(version, microService)
		if err != nil {
			log.Error(err, "Make Deployment for version error", "versionName", version.Name)
			return err
		}
		if err := controllerutil.SetControllerReference(microService, deploy, r.scheme); err != nil {
			log.Error(err, "Set DeployVersion CtlRef Error", "versionName", version.Name)
			return err
		}

		newDeploys[deploy.Name] = deploy
		found := &appsv1.Deployment{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)

		if err != nil && errors.IsNotFound(err) {

			log.Info("Old Deployment NotFound and Creating new one", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Create(context.TODO(), deploy)
			return err

		} else if err != nil {

			log.Error(err, "Get Deployment info Error", "namespace", deploy.Namespace, "name", deploy.Name)
			return err

		} else if !reflect.DeepEqual(deploy.Spec, found.Spec) {

			// Update the found object and write the result back if there are any changes
			found.Spec = deploy.Spec
			log.Info("Old deployment changed and Updating Deployment to reconcile", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}

		}
	}
	return r.cleanUpDeploy(microService, newDeploys)
}

func makeVersionDeployment(version *appv1.DeployVersion, microService *appv1.MicroService) (*appsv1.Deployment, error) {

	labels := microService.Labels
	if labels == nil {
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

func (r *ReconcileMicroService) cleanUpDeploy(microService *appv1.MicroService, newDeployList map[string]*appsv1.Deployment) error {
	// Check if the MicroService not exists
	ctx := context.Background()

	deployList := appsv1.DeploymentList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/service"] = microService.Name

	if err := r.List(ctx, client.InNamespace(microService.Namespace).
		MatchingLabels(labels), &deployList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return err
	}

	for _, oldDeploy := range deployList.Items {
		if _, exist := newDeployList[oldDeploy.Name]; exist == false {
			log.Info("Find orphan Deployment", "namespace", microService.Namespace, "MicroService", microService.Name, "Deployment", oldDeploy.Name)
			err := r.Delete(context.TODO(), &oldDeploy)
			if err != nil {
				log.Error(err, "Delete orphan Deployment error", "namespace", oldDeploy.Namespace, "name", oldDeploy.Name)
				return err
			}
		}
	}
	return nil
}
