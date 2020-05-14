/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	// "time"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	// "k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	"reflect"
	batchv1alpha1 "sdewan.akraino.org/sdewan/api/v1alpha1"
	// "sdewan.akraino.org/sdewan/cnfprovider"
	"sdewan.akraino.org/sdewan/openwrt"
	"strconv"
)

type Mwan3PolicyHandler struct {
}

func (m *Mwan3PolicyHandler) GetType() string {
	return "Mwan3Policy"
}

func (m *Mwan3PolicyHandler) GetName(instance runtime.Object) string {
	policy := instance.(*batchv1alpha1.Mwan3Policy)
	return policy.Name
}

func (m *Mwan3PolicyHandler) GetFinalizer() string {
	return "rule.finalizers.sdewan.akraino.org"
}

func (m *Mwan3PolicyHandler) GetInstance(r client.Client, ctx context.Context, req ctrl.Request) (runtime.Object, error) {
	instance := &batchv1alpha1.Mwan3Policy{}
	err := r.Get(ctx, req.NamespacedName, instance)
	return instance, err
}

func (m *Mwan3PolicyHandler) Convert(instance runtime.Object, deployment appsv1.Deployment) (openwrt.IOpenWrtObject, error) {
	policy := instance.(*batchv1alpha1.Mwan3Policy)
	members := make([]openwrt.SdewanMember, len(policy.Spec.Members))
	for i, membercr := range policy.Spec.Members {
		iface, err := net2iface(membercr.Network, deployment)
		if err != nil {
			return nil, err
		}
		members[i] = openwrt.SdewanMember{
			Interface: iface,
			Metric:    strconv.Itoa(membercr.Metric),
			Weight:    strconv.Itoa(membercr.Weight),
		}
	}
	return &openwrt.SdewanPolicy{Name: policy.Name, Members: members}, nil
}

func printjson(word openwrt.IOpenWrtObject) {
	policy_obj, _ := json.Marshal(word)
	fmt.Printf("%s\n", string(policy_obj))
}

func (m *Mwan3PolicyHandler) IsEqual(instance1 openwrt.IOpenWrtObject, instance2 openwrt.IOpenWrtObject) bool {
	policy1 := instance1.(*openwrt.SdewanPolicy)
	policy2 := instance2.(*openwrt.SdewanPolicy)
	return reflect.DeepEqual(*policy1, *policy2)
}

func (m *Mwan3PolicyHandler) GetObject(clientInfo *openwrt.OpenwrtClientInfo, name string) (openwrt.IOpenWrtObject, error) {
	openwrtClient := openwrt.GetOpenwrtClient(*clientInfo)
	mwan3 := openwrt.Mwan3Client{OpenwrtClient: openwrtClient}
	ret, err := mwan3.GetPolicy(name)
	printjson(ret)
	return ret, err
}

func (m *Mwan3PolicyHandler) CreateObject(clientInfo *openwrt.OpenwrtClientInfo, instance openwrt.IOpenWrtObject) (openwrt.IOpenWrtObject, error) {
	openwrtClient := openwrt.GetOpenwrtClient(*clientInfo)
	mwan3 := openwrt.Mwan3Client{OpenwrtClient: openwrtClient}
	policy := instance.(*openwrt.SdewanPolicy)
	return mwan3.CreatePolicy(*policy)
}

func (m *Mwan3PolicyHandler) UpdateObject(clientInfo *openwrt.OpenwrtClientInfo, instance openwrt.IOpenWrtObject) (openwrt.IOpenWrtObject, error) {
	openwrtClient := openwrt.GetOpenwrtClient(*clientInfo)
	mwan3 := openwrt.Mwan3Client{OpenwrtClient: openwrtClient}
	policy := instance.(*openwrt.SdewanPolicy)
	return mwan3.UpdatePolicy(*policy)
}

func (m *Mwan3PolicyHandler) DeleteObject(clientInfo *openwrt.OpenwrtClientInfo, name string) error {
	openwrtClient := openwrt.GetOpenwrtClient(*clientInfo)
	mwan3 := openwrt.Mwan3Client{OpenwrtClient: openwrtClient}
	return mwan3.DeletePolicy(name)
}

func (m *Mwan3PolicyHandler) Restart(clientInfo *openwrt.OpenwrtClientInfo) (bool, error) {
	openwrtClient := openwrt.GetOpenwrtClient(*clientInfo)
	service := openwrt.ServiceClient{OpenwrtClient: openwrtClient}
	return service.ExecuteService("mwan3", "restart")
}

// Mwan3PolicyReconciler reconciles a Mwan3Policy object
type Mwan3PolicyReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.sdewan.akraino.org,resources=mwan3policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.sdewan.akraino.org,resources=mwan3policies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
func (r *Mwan3PolicyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return ProcessReconcile(r, r.Log, req, &Mwan3PolicyHandler{})

	/*
	   func (r *Mwan3PolicyReconciler) OldReconcile (req ctrl.Request) (ctrl.Result, error) {
	   	ctx := context.Background()
	   	log := r.Log.WithValues("mwan3policy", req.NamespacedName)

	   	// your logic here
	   	during, _ := time.ParseDuration("5s")
	   	instance := &batchv1alpha1.Mwan3Policy{}
	   	err := r.Get(ctx, req.NamespacedName, instance)
	   	if err != nil {
	   		if errors.IsNotFound(err) {
	   			// No instance
	   			return ctrl.Result{}, nil
	   		}
	   		// Error reading the object - requeue the request.
	   		return ctrl.Result{RequeueAfter: during}, nil
	   	}
	   	cnf, err := cnfprovider.NewWrt(req.NamespacedName.Namespace, instance.Labels["sdewanPurpose"], r.Client)
	   	if err != nil {
	   		log.Error(err, "Failed to get cnf")
	   		// A new event are supposed to be received upon cnf ready
	   		// so not requeue
	   		return ctrl.Result{}, nil
	   	}
	   	finalizerName := "rule.finalizers.sdewan.akraino.org"
	   	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
	   		// creating or updating CR
	   		if cnf == nil {
	   			// no cnf exists
	   			log.Info("No cnf exist, so not create/update mwan3 policy")
	   			return ctrl.Result{}, nil
	   		}
	   		changed, err := cnf.AddUpdateMwan3Policy(instance)
	   		if err != nil {
	   			log.Error(err, "Failed to add/update mwan3 policy")
	   			return ctrl.Result{RequeueAfter: during}, nil
	   		}
	   		if !containsString(instance.ObjectMeta.Finalizers, finalizerName) {
	   			log.Info("Adding finalizer for mwan3 policy")
	   			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
	   			if err := r.Update(ctx, instance); err != nil {
	   				return ctrl.Result{}, err
	   			}
	   		}
	   		if changed {
	   			instance.Status.AppliedVersion = instance.ResourceVersion
	   			instance.Status.AppliedTime = &metav1.Time{Time: time.Now()}
	   			instance.Status.InSync = true
	   			err = r.Status().Update(ctx, instance)
	   			if err != nil {
	   				log.Error(err, "Failed to update mwan3 policy status")
	   				return ctrl.Result{}, err
	   			}
	   		}
	   	} else {
	   		// deletin CR
	   		if cnf == nil {
	   			// no cnf exists
	   			if containsString(instance.ObjectMeta.Finalizers, finalizerName) {
	   				instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
	   				if err := r.Update(ctx, instance); err != nil {
	   					return ctrl.Result{}, err
	   				}
	   			}
	   			return ctrl.Result{}, nil
	   		}
	   		_, err := cnf.DeleteMwan3Policy(instance)
	   		if err != nil {
	   			log.Error(err, "Failed to delete mwan3 policy")
	   			return ctrl.Result{RequeueAfter: during}, nil
	   		}
	   		if containsString(instance.ObjectMeta.Finalizers, finalizerName) {
	   			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
	   			if err := r.Update(ctx, instance); err != nil {
	   				return ctrl.Result{}, err
	   			}
	   		}
	   	}

	   	return ctrl.Result{}, nil
	*/
}

func (r *Mwan3PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.Mwan3Policy{}).
		Complete(r)
}

/*
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
*/
