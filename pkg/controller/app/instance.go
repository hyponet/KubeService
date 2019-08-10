package app

import (
	appv1 "KubeService/pkg/apis/app/v1"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

func (r *ReconcileApp) reconcileMicroService(app *appv1.App) error {
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
			if err = r.Create(context.TODO(), ms); err != nil {
				return err
			}
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

func (r *ReconcileApp) syncAppStatus(app *appv1.App) error {
	ctx := context.Background()
	// app 状态同步到位
	if app.Status.AvailableMicroServices != 0 && app.Status.AvailableMicroServices == app.Status.TotalMicroServices {
		app.Status.FromManager = appv1.ManagerNone
		err := r.Status().Update(ctx, app)
		return err
	}

	newStatus, err := r.calculateStatus(app)
	if err != nil {
		return err
	}

	condType := appv1.AppProgressing
	status := appv1.ConditionTrue
	reason := ""
	message := ""
	if newStatus.AvailableMicroServices == newStatus.TotalMicroServices {
		condType = appv1.AppAvailable
		reason = "All deploy have updated."
	} else if newStatus.AvailableMicroServices > newStatus.TotalMicroServices {
		reason = "Some microservices got to be deleted."
	} else {
		reason = "Some microservices got to be created."
	}
	condition := appv1.AppCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	conditions := app.Status.Conditions
	for i := range conditions {
		newStatus.Conditions = append(newStatus.Conditions, conditions[i])
	}
	newStatus.Conditions = append(newStatus.Conditions, condition)
	app.Status = newStatus
	err = r.Status().Update(ctx, app)
	return err
}

func (r *ReconcileApp) calculateStatus(app *appv1.App) (appv1.AppStatus, error) {
	// Check if the MicroService not exists
	ctx := context.Background()

	msList := appv1.MicroServiceList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/app"] = app.Name

	al := int32(len(msList.Items))
	tl := int32(len(app.Spec.MicroServices))
	newStatus := appv1.AppStatus{
		AvailableMicroServices: al,
		TotalMicroServices:     tl,
	}
	if err := r.List(ctx, client.InNamespace(app.Namespace).
		MatchingLabels(labels), &msList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return newStatus, err
	}
	newStatus.AvailableMicroServices = int32(len(msList.Items))

	return newStatus, nil
}

func (r *ReconcileApp) reconcileAppForMutilCluster(app *appv1.App) error {
	status := app.Status
	if status.FromManager == appv1.ManagerCreated || status.FromManager == appv1.ManagerUpdated{
		return nil
	}
	var netTransport = &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	var netClient = &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}

	clusterName := os.Getenv("cluster_name")
	m := os.Getenv("manager_url")
	managerUrl := m + "/apps"
	yml, err := json.Marshal(app)
	if err != nil {
		return nil
	}

	data := map[string]string{"cluster_name": clusterName, "yml": string(yml)}
	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	response, err := netClient.Post(
		managerUrl,
		"application/json",
		bytes.NewBuffer(dataJson),
	)
	if err != nil {
		return nil
	}
	err = response.Body.Close()
	if err != nil {
		return nil
	}

	if response.StatusCode == 200 {
		body, err := ioutil.ReadAll(response.Body)
		fmt.Println(string(body))
		if err != nil {
			return nil
		}
	}
	return nil
}
