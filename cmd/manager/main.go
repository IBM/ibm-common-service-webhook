//
// Copyright 2022 IBM Corporation
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

package main

import (
	"os"
	"runtime"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"

	apisv1alpha1 "github.com/IBM/ibm-common-service-webhook/pkg/apis/v1alpha1"
	"github.com/IBM/ibm-common-service-webhook/pkg/controller/nsmappingconfigmap"
	"github.com/IBM/ibm-common-service-webhook/pkg/controller/operandrequest"
	"github.com/IBM/ibm-common-service-webhook/pkg/controller/podpreset"
	"github.com/IBM/ibm-common-service-webhook/pkg/utils"
	"github.com/IBM/ibm-common-service-webhook/pkg/webhooks"
	"github.com/IBM/ibm-common-service-webhook/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func printVersion() {
	klog.Infof("Operator Version: %s", version.Version)
	klog.Infof("Go Version: %s", runtime.Version())
	klog.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
}

var (
	scheme = k8sruntime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(odlmv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {

	klog.InitFlags(nil)
	defer klog.Flush()

	printVersion()

	namespace := utils.GetWatchNamespace()
	options := ctrl.Options{
		Scheme:    scheme,
		Namespace: namespace,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		klog.Errorf("unable to start manager: %v", err)
		os.Exit(1)
	}

	if err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	klog.Info("Registering Components.")

	if err = (&podpreset.ReconcilePodPreset{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to create controller: %v", err)
		os.Exit(1)
	}

	// Start up the webhook server
	if err := setupWebhooks(mgr, namespace); err != nil {
		klog.Error(err, "Error setting up webhook server")
	}

	klog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func setupWebhooks(mgr manager.Manager, namespace string) error {

	klog.Info("Creating common service webhook configuration")

	if utils.GetEnableOpreqWebhook() {
		webhooks.Config.AddWebhook(webhooks.CSWebhook{
			Name:        "ibm-operandrequest-webhook-configuration",
			WebhookName: "ibm-cloudpak-operandrequest.operator.ibm.com",
			Rule: webhooks.NewRule().
				OneResource("operator.ibm.com", "v1alpha1", "operandrequests").
				ForUpdate().
				ForCreate().
				NamespacedScope(),
			Register: webhooks.AdmissionWebhookRegister{
				Type: webhooks.MutatingType,
				Path: "/mutate-ibm-cp-operandrequest",
				Hook: &admission.Webhook{
					Handler: &operandrequest.Mutator{
						Reader: mgr.GetAPIReader(),
					},
				},
			},
		})
		webhooks.Config.AddWebhook(webhooks.CSWebhook{
			Name:        "ibm-cs-ns-mapping-webhook-configuration",
			WebhookName: "cs-ns-mapping-configmap.operator.ibm.com",
			Rule: webhooks.NewRule().
				OneResource("", "v1", "configmaps").
				ForUpdate().
				ForCreate().
				NamespacedScope(),
			Register: webhooks.AdmissionWebhookRegister{
				Type: webhooks.ValidatingType,
				Path: "/validate-ibm-cs-ns-map",
				Hook: &admission.Webhook{
					Handler: &nsmappingconfigmap.Mutator{
						Reader: mgr.GetAPIReader(),
					},
				},
			},
			NsSelector: v1.LabelSelector{
				MatchExpressions: []v1.LabelSelectorRequirement{
					{
						Key:      "kubernetes.io/metadata.name",
						Operator: v1.LabelSelectorOpIn,
						Values: []string{
							"kube-public",
						},
					},
				},
			},
		})
	}

	klog.Info("setting up webhook server")
	if err := webhooks.Config.SetupServer(mgr, namespace); err != nil {
		return err
	}

	return nil
}
