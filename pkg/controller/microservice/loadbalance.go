package microservice

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"context"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileMicroService) reconcileLoadBalance(microService *appv1.MicroService) error {
	lb := microService.Spec.LoadBalance
	if lb == nil {
		return nil
	}

	if len(microService.Spec.Versions) == 0 {
		return nil
	}

	currentVersion := microService.Spec.Versions[0]
	for _, version := range microService.Spec.Versions {
		if version.Name == microService.Spec.CurrentVersionName {
			currentVersion = version
			break
		}
	}

	enableSVC := false
	if lb.Service != nil {
		svcLB := lb.Service
		// If use define custom Service
		enableSVC = true
		if svcLB.Spec.Selector == nil {
			svcLB.Spec.Selector = currentVersion.Template.Selector.MatchLabels
		}
		svc, err := makeService(microService.Name, microService.Namespace, microService.Labels, &svcLB.Spec)
		if err != nil {
			return err
		}
		svc.Labels = microService.Labels
		if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
			return err
		}

		if err := r.updateOrCreateSvc(svc); err != nil {
			return err
		}
	}

	if lb.Ingress != nil {
		ingressLB := lb.Ingress
		ingress := &extensionsv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ingressLB.Name,
				Namespace: microService.Namespace,
				Labels:    microService.Labels,
			},
			Spec: ingressLB.Spec,
		}
		if err := controllerutil.SetControllerReference(microService, ingress, r.scheme); err != nil {
			return err
		}
		found := &extensionsv1beta1.Ingress{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating Ingress", "namespace", ingress.Namespace, "name", ingress.Name)
			if err = r.Create(context.TODO(), ingress); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if !reflect.DeepEqual(ingress.Spec, found.Spec) {
			log.Info("Find Ingress as been modified, but not reconciled", "namespace", ingress.Namespace, "name", ingress.Name)
		}
	}

	if !enableSVC {
		return nil
	}
	for _, version := range microService.Spec.Versions {
		spec := lb.Service.Spec.DeepCopy()
		spec.Selector = version.Template.Selector.MatchLabels
		svc, err := makeService(microService.Name+"-"+version.Name, microService.Namespace, microService.Labels, spec)
		if err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
			return err
		}

		if err := r.updateOrCreateSvc(svc); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileMicroService) updateOrCreateSvc(svc *v1.Service) error {
	// Check if the Service already exists
	found := &v1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "namespace", svc.Namespace, "name", svc.Name)
		err = r.Create(context.TODO(), svc)
		return err
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(svc.Spec, found.Spec) {
		log.Info("Find SVC as been modified, but not reconciled", "namespace", svc.Namespace, "name", svc.Name)
	}
	return nil
}

func makeService(name string, namespace string, label map[string]string, svcSpec *v1.ServiceSpec) (*v1.Service, error) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    label,
		},
		Spec: *svcSpec,
	}
	return svc, nil
}
