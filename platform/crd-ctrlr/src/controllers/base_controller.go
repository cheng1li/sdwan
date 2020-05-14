package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	errs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	// "sdewan.akraino.org/sdewan/openwrt"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1alpha1 "sdewan.akraino.org/sdewan/api/v1alpha1"
	"sdewan.akraino.org/sdewan/basehandler"
	"sdewan.akraino.org/sdewan/cnfprovider"
)

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func getPurpose(instance runtime.Object) string {
	value := reflect.ValueOf(instance)
	field := reflect.Indirect(value).FieldByName("Labels")
	labels := field.Interface().(map[string]string)
	return labels["sdewanPurpose"]
}

func getDeletionTempstamp(instance runtime.Object) *metav1.Time {
	// to do: time.Time
	value := reflect.ValueOf(instance)
	field := reflect.Indirect(value).FieldByName("DeletionTimestamp")
	return field.Interface().(*metav1.Time)
}

func getFinalizers(instance runtime.Object) []string {
	value := reflect.ValueOf(instance)
	field := reflect.Indirect(value).FieldByName("Finalizers")
	return field.Interface().([]string)
}

func setStatus(instance runtime.Object, t *metav1.Time, isSync bool) {
	value := reflect.ValueOf(instance)
	field_rv := reflect.Indirect(value).FieldByName("ResourceVersion")
	rv := field_rv.Interface().(string)
	field_status := reflect.Indirect(value).FieldByName("Status")
	status := field_status.Interface().(batchv1alpha1.SdewanStatus) //undefined: SdewanStatus
	status.AppliedVersion = rv
	status.AppliedTime = t
	status.InSync = isSync
	field_status.Set(reflect.ValueOf(status))
}

func appendFinalizer(instance runtime.Object, item string) {
	// to do: ObjectMeta
	value := reflect.ValueOf(instance)
	field := reflect.Indirect(value).FieldByName("ObjectMeta")
	base_obj := field.Interface().(metav1.ObjectMeta) //  undefined: ObjectMeta
	base_obj.Finalizers = append(base_obj.Finalizers, item)
	field.Set(reflect.ValueOf(base_obj))
}

func removeFinalizer(instance runtime.Object, item string) {
	value := reflect.ValueOf(instance)
	field := reflect.Indirect(value).FieldByName("ObjectMeta")
	base_obj := field.Interface().(metav1.ObjectMeta) //  undefined: ObjectMeta
	base_obj.Finalizers = removeString(base_obj.Finalizers, item)
	field.Set(reflect.ValueOf(base_obj))
}

func net2iface(net string, deployment appsv1.Deployment) (string, error) {
	type Iface struct {
		DefaultGateway bool `json:"defaultGateway,string"`
		Interface      string
		Name           string
	}
	type NfnNet struct {
		Type      string
		Interface []Iface
	}
	ann := deployment.Spec.Template.Annotations
	nfnNet := NfnNet{}
	err := json.Unmarshal([]byte(ann["k8s.plugin.opnfv.org/nfn-network"]), &nfnNet)
	if err != nil {
		return "", err
	}
	for _, iface := range nfnNet.Interface {
		if iface.Name == net {
			return iface.Interface, nil
		}
	}
	return "", errors.New(fmt.Sprintf("No matched network in annotation: %s", net)) //debug undefined: "k8s.io/apimachinery/pkg/api/errors".New

}

// Common Reconcile Processing
func ProcessReconcile(r client.Client, logger logr.Logger, req ctrl.Request, handler basehandler.ISdewanHandler) (ctrl.Result, error) {
	ctx := context.Background()
	log := logger.WithValues(handler.GetType(), req.NamespacedName)

	// your logic here
	during, _ := time.ParseDuration("5s")

	//instance := &batchv1alpha1.Mwan3Policy{}
	//err := r.Get(ctx, req.NamespacedName, instance)
	instance, err := handler.GetInstance(r, ctx, req)
	if err != nil {
		if errs.IsNotFound(err) {
			// No instance
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: during}, nil
	}
	// cnf, err := cnfprovider.NewWrt(req.NamespacedName.Namespace, instance.Labels["sdewanPurpose"], r.Client)
	// Labels: map[string]string
	purpose := getPurpose(instance)
	cnf, err := cnfprovider.NewOpenWrt(req.NamespacedName.Namespace, purpose, r)
	if err != nil {
		log.Error(err, "Failed to get cnf")
		// A new event are supposed to be received upon cnf ready
		// so not requeue
		return ctrl.Result{}, nil
	}
	// finalizerName := "rule.finalizers.sdewan.akraino.org"
	finalizerName := handler.GetFinalizer()
	// if instance.ObjectMeta.DeletionTimestamp.IsZero() {
	// DeletionTimestamp: *Time
	delete_timestamp := getDeletionTempstamp(instance)

	if delete_timestamp.IsZero() {
		fmt.Printf("file: base controller.go\n line:154\n-------------create update CR\n")
		// creating or updating CR
		if cnf == nil {
			// no cnf exists
			log.Info("No cnf exist, so not create/update " + handler.GetType())
			return ctrl.Result{}, nil
		}
		fmt.Printf("file: base controller.go\n line:184\n-------------update cr to cnf------- CR\n")
		changed, err := cnf.AddOrUpdateObject(handler, instance)
		if err != nil {
			log.Error(err, "Failed to add/update "+handler.GetType())
			return ctrl.Result{RequeueAfter: during}, nil
		}
		// if !containsString(instance.ObjectMeta.Finalizers, finalizerName) {
		// Finalizers: []string
		finalizers := getFinalizers(instance)
		if !containsString(finalizers, finalizerName) {
			log.Info("Adding finalizer for " + handler.GetType())
			// instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
			// Finalizers: []string
			appendFinalizer(instance, finalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		if changed {
			fmt.Printf("file: base controller.go\n line:184\n-------------cnf changed ------- CR\n")
			// instance.Status.AppliedVersion = instance.ResourceVersion
			// instance.Status.AppliedTime = &metav1.Time{Time: time.Now()}
			// instance.Status.InSync = true
			// Status: SdewanStatus
			fmt.Println("+instance++++++++++++++")
			fmt.Println(instance)
			setStatus(instance, &metav1.Time{Time: time.Now()}, true)

			fmt.Printf("file: base controller.go\n line:184\n-------------set status ------- CR\n")
			err = r.Status().Update(ctx, instance)
			if err != nil {
				log.Error(err, "Failed to update status for "+handler.GetType())
				return ctrl.Result{}, err
			}
		}
	} else {
		// deletin CR
		fmt.Printf("file: base controller.go\n line:193\n-------------delete cr------- CR\n")
		if cnf == nil {
			// no cnf exists
			finalizers := getFinalizers(instance)
			if containsString(finalizers, finalizerName) {
				// instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
				removeFinalizer(instance, finalizerName)
				if err := r.Update(ctx, instance); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		}
		//_, err := cnf.DeleteMwan3Policy(instance)
		fmt.Printf("file: base controller.go\n line:193\n-------------delete cnf policy------- CR\n")
		_, err := cnf.DeleteObject(handler, instance)
		fmt.Printf("file: base controller.go\n line 210\n -------------type of error from cnf.DeleteObject %T", err)
		fmt.Printf("file: base controller.go\n line 210\n -------------value of error from cnf.DeleteObject %v", err)

		// labels := field.Interface().(map[string]string)
		// return labels["sdewanPurpose"]
		// fmt.Printf("delete response  type is %T \n", *err)
		// fmt.Printf("delete response  type is %v \n", *err)
		// json.Unmarshal([]byte(
		if err != nil {
			value := reflect.ValueOf(err)
			err_rv := reflect.Indirect(value).FieldByName("Code")
			err_code := err_rv.Interface().(int)
			// fmt.Printf("delete response type is %T \n", err_code)
			// fmt.Printf("delete response value is %v \n", err_code)
			if err_code == 404 {
				// if containsString(instance.ObjectMeta.Finalizers, finalizerName) {
				finalizers := getFinalizers(instance)
				if containsString(finalizers, finalizerName) {
					// instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
					removeFinalizer(instance, finalizerName)
					if err := r.Update(ctx, instance); err != nil {
						return ctrl.Result{}, err
					}
				}
			}
			log.Error(err, "Failed to delete "+handler.GetType())
			return ctrl.Result{RequeueAfter: during}, nil
		}
		// if containsString(instance.ObjectMeta.Finalizers, finalizerName) {
		finalizers := getFinalizers(instance)
		if containsString(finalizers, finalizerName) {
			// instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
			removeFinalizer(instance, finalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
