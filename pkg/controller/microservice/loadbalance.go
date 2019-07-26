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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

func (r *ReconcileMicroService) reconcileLoadBalance(microService *appv1.MicroService) error {
	lb := microService.Spec.LoadBalance
	staySVCName := make([]string, 5)
	stayIngressName := make([]string, 5)

	if lb == nil {
		return r.clearUpLB(microService, &staySVCName, &stayIngressName)
	}

	if len(microService.Spec.Versions) == 0 {
		return r.clearUpLB(microService, &staySVCName, &stayIngressName)
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
		svcLB.Spec.Selector = currentVersion.Template.Selector.MatchLabels
		svc, err := makeService(microService.Name, microService.Namespace, microService.Labels, &svcLB.Spec)
		if err != nil {
			return err
		}
		svc.Labels = microService.Labels
		if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
			return err
		}

		if err := r.updateOrCreateSVC(svc); err != nil {
			return err
		}
		staySVCName = append(staySVCName, svc.Name)
	}

	enableIngress := false
	if lb.Ingress != nil {
		enableIngress = true
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
		if err := r.updateOrCreateIngress(ingress); err != nil {
			return err
		}
		stayIngressName = append(stayIngressName, ingress.Name)
	}

	if enableSVC {
		for _, version := range microService.Spec.Versions {
			spec := lb.Service.Spec.DeepCopy()
			spec.Selector = version.Template.Selector.MatchLabels
			serviceName := version.ServiceName
			if serviceName == "" {
				serviceName = microService.Name + "-" + version.Name
			}
			svc, err := makeService(serviceName, microService.Namespace, microService.Labels, spec)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
				return err
			}

			if err := r.updateOrCreateSVC(svc); err != nil {
				return err
			}
			version.ServiceName = serviceName
			staySVCName = append(staySVCName, serviceName)
		}
	}

	if enableIngress {
		for _, version := range microService.Spec.Versions {
			if version.Canary == nil {
				continue
			}
			ingress, err := makeCanaryIngress(microService, &lb.Ingress.Spec, &version)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(microService, ingress, r.scheme); err != nil {
				return err
			}
			if err := r.updateOrCreateIngress(ingress); err != nil {
				return err
			}
			stayIngressName = append(stayIngressName, ingress.Name)
		}
	}

	if err := r.Update(context.TODO(), microService); err != nil {
		return err
	}

	return r.clearUpLB(microService, &staySVCName, &stayIngressName)
}

func (r *ReconcileMicroService) updateOrCreateSVC(svc *v1.Service) error {
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
		svc.Spec.ClusterIP = found.Spec.ClusterIP
		found.Spec = svc.Spec
		if err = r.Update(context.TODO(), found); err != nil {
			return err
		}
		log.Info("Find SVC as been modified, update", "namespace", svc.Namespace, "name", svc.Name)
	}
	return nil
}
func (r *ReconcileMicroService) updateOrCreateIngress(ingress *extensionsv1beta1.Ingress) error {
	found := &extensionsv1beta1.Ingress{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Ingress", "namespace", ingress.Namespace, "name", ingress.Name)
		if err = r.Create(context.TODO(), ingress); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(ingress.Spec, found.Spec) || !reflect.DeepEqual(ingress.Annotations, found.Annotations) {
		found.Spec = ingress.Spec
		found.Annotations = ingress.Annotations
		if err = r.Update(context.TODO(), found); err != nil {
			return err
		}
		log.Info("Find Ingress as been modified", "namespace", ingress.Namespace, "name", ingress.Name)
	}
	return nil
}

func (r *ReconcileMicroService) clearUpLB(microService *appv1.MicroService, staySVCName *[]string, stayIngressName *[]string) error {
	opts := &client.ListOptions{}
	opts.InNamespace(microService.Namespace)
	opts.MatchingLabels(map[string]string{"app.o0w0o.cn/service": microService.Name})

	allSVC := &v1.ServiceList{}
	if err := r.List(context.TODO(), opts, allSVC); err != nil {
		return err
	}
	for _, svc := range allSVC.Items {
		found := false
		for _, svcName := range *staySVCName {
			if svcName == svc.Name {
				found = true
				break
			}
		}
		if !found {
			if err := r.Client.Delete(context.TODO(), svc.DeepCopy()); err != nil {
				return err
			}
		}
	}

	allIngress := &extensionsv1beta1.IngressList{}
	if err := r.List(context.TODO(), opts, allIngress); err != nil {
		return err
	}
	for _, ingress := range allIngress.Items {
		found := false
		for _, ingressName := range *stayIngressName {
			if ingressName == ingress.Name {
				found = true
				break
			}
		}
		if !found {
			if err := r.Client.Delete(context.TODO(), ingress.DeepCopy()); err != nil {
				return err
			}
		}
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

func makeCanaryIngress(microService *appv1.MicroService, ingressSpec *extensionsv1beta1.IngressSpec, version *appv1.DeployVersion) (*extensionsv1beta1.Ingress, error) {
	// TODO nginx ingress controller support ONLY now
	canary := version.Canary
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/canary":        "true",
		"nginx.ingress.kubernetes.io/canary-weight": strconv.Itoa(canary.Weight),
	}

	if canary.Header != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-header"] = canary.Header
		annotations["nginx.ingress.kubernetes.io/canary-by-header-value"] = canary.HeaderValue
	}

	if canary.Cookie != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-cookie"] = canary.Cookie
	}

	if canary.CanaryIngressName == "" {
		canary.CanaryIngressName = microService.Name + "-" + version.Name + "-canary"
	}

	ingressSpec = ingressSpec.DeepCopy()

	if ingressSpec.Rules != nil {
		for i, rule := range ingressSpec.Rules {
			if rule.IngressRuleValue.HTTP == nil {
				continue
			}
			for j, path := range rule.IngressRuleValue.HTTP.Paths {
				if path.Backend.ServiceName == microService.Spec.LoadBalance.Service.Name {
					ingressSpec.Rules[i].IngressRuleValue.HTTP.Paths[j].Backend.ServiceName = version.ServiceName
				}
			}
		}
	}
	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        canary.CanaryIngressName,
			Namespace:   microService.Namespace,
			Labels:      microService.Labels,
			Annotations: annotations,
		},
		Spec: *ingressSpec,
	}

	return ingress, nil
}
