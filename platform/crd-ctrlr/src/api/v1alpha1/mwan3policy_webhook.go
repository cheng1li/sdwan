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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var mwan3policylog = logf.Log.WithName("mwan3policy-resource")

func (r *Mwan3Policy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-batch-sdewan-akraino-org-v1alpha1-mwan3policy,mutating=false,failurePolicy=fail,groups=batch.sdewan.akraino.org,resources=mwan3policies,versions=v1alpha1,name=vmwan3policy.kb.io

var _ webhook.Validator = &Mwan3Policy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Mwan3Policy) ValidateCreate() error {
	mwan3policylog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Mwan3Policy) ValidateUpdate(old runtime.Object) error {
	mwan3policylog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Mwan3Policy) ValidateDelete() error {
	mwan3policylog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

////////////////////////////////////////////////////////////////////////////////////
func (r *Mwan3Policy) SetupWebhookWithManager2(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register("/validate-v1-pod", &webhook.Admission{Handler: &podValidator{Client: mgr.GetClient()}})
	return nil
}

// +kubebuilder:webhook:path=/validate-v1-pod,mutating=false,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kb.io

// podValidator validates Pods
type podValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// podValidator admits a pod iff a specific annotation exists.
func (v *podValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Kind.Group != "batch.sdewan.akraino.org" {
		return admission.Errored(http.StatusBadRequest, errors.New("The group is not batch.sdewan.akraino.org"))
	}
	var meta metav1.ObjectMeta
	var err error
	var obj runtime.Object
	switch req.Kind.Kind {
	case "Mwan3Policy":
		obj = &Mwan3Policy{}
	default:
		return admission.Errored(http.StatusBadRequest, errors.New(fmt.Sprintf("Kind is not supported: %v", req.Kind)))
	}

	switch req.Operation {
	case "CREATE", "UPDATE":
		err = v.decoder.Decode(req, obj)
	case "DELETE":
		err = v.Client.Get(context.Background(), types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, obj)
	}
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// objectmeta is the second field in Object, so Field(1)
	meta = reflect.ValueOf(obj).Elem().Field(1).Interface().(metav1.ObjectMeta)
	sdewanPurpose := meta.Labels["sdewanPurpose"]
	if sdewanPurpose == "" {
		return admission.Allowed("")
	}
	userRolePers := getSdewanPermission(v.Client, req.UserInfo)
	rolePer := map[string][]string{"mwanpolicies": {"app-intent"}}
	resourcePer := rolePer[req.Resource.Resource]
	if resourcePer != nil {
		for _, p := range resourcePer {
			if p == sdewanPurpose {
				return admission.Allowed("")
			}
		}
	}
	return admission.Denied("Your roles don't have the permission")
}

type SdewanpurposeRole map[string][]string

func getSdewanPermission(c client.Client, userInfo authenticationv1.UserInfo) []SdewanpurposeRole {
	ServiceAccount := false
	for group := range userInfo.Groups {
		if group == "system:serviceaccounts" {
			ServiceAccount = true
			break
		}
	}
}
// podValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *podValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
