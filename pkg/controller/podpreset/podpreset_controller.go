//
// Copyright 2021 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package podpreset

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/IBM/ibm-common-service-webhook/pkg/apis/v1alpha1"
	"github.com/IBM/ibm-common-service-webhook/pkg/utils"
	"github.com/IBM/ibm-common-service-webhook/pkg/webhooks"
)

const (
	podpresetName = "cs-podpreset.operator.ibm.com"
)

// ReconcilePodPreset reconciles a PodPreset object
type ReconcilePodPreset struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PodPreset object and makes changes based on the state read
// and what is in the PodPreset.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePodPreset) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling PodPreset %s/%s", request.Namespace, request.Name)

	ns := &corev1.Namespace{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: request.Namespace}, ns)
	if err != nil {
		return ctrl.Result{}, err
	}

	if utils.GetEnableOpreqWebhook() {
		if err := r.AddNameLabeltoNs("kube-public"); err != nil {
			klog.Error(err, "Failed to add label to namespace kube-public")
			return ctrl.Result{}, err
		}
	}

	currentLabels := ns.GetLabels()
	if len(currentLabels) == 0 {
		ns.SetLabels(map[string]string{
			"managed-by-common-service-webhook": "true",
		})
	} else {
		currentLabels["managed-by-common-service-webhook"] = "true"
		ns.SetLabels(currentLabels)
	}

	if err := r.Client.Update(context.TODO(), ns); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the PodPreset instance
	instance := &operatorv1alpha1.PodPreset{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Reconcile the webhooks
	if err := webhooks.Config.Reconcile(context.TODO(), r.Client, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReconcilePodPreset) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.PodPreset{}).
		Complete(r)
}

func (r *ReconcilePodPreset) AddNameLabeltoNs(nsName string) error {
	ns := &corev1.Namespace{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: nsName}, ns)
	if err != nil {
		klog.Error(err)
		return err
	}

	if _, ok := ns.GetLabels()["kubernetes.io/metadata.name"]; ok {
		return nil
	}

	ns.SetLabels(map[string]string{
		"kubernetes.io/metadata.name": nsName,
	})

	if err := r.Client.Update(context.TODO(), ns); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}
