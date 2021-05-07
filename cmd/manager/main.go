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

package main

import (
	"os"
	"runtime"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/IBM/ibm-common-service-webhook/pkg/apis"
	"github.com/IBM/ibm-common-service-webhook/pkg/controller"
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

func main() {

	klog.InitFlags(nil)
	defer klog.Flush()

	printVersion()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	namespace := utils.GetWatchNamespace()
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace: namespace,
	})

	if err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	klog.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		klog.Error(err, "")
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
	managedbyCSWebhookLabel := make(map[string]string)
	managedbyCSWebhookLabel["managed-by-common-service-webhook"] = "true"
	managedbyCSSelector := v1.LabelSelector{
		MatchLabels: managedbyCSWebhookLabel,
	}
	webhooks.Config.AddWebhook(webhooks.CSWebhook{
		Name:        "ibm-common-service-webhook-configuration",
		WebhookName: "cs-podpreset.operator.ibm.com",
		Rule: webhooks.NewRule().
			OneResource("", "v1", "pods").
			ForUpdate().
			ForCreate().
			NamespacedScope(),
		Register: webhooks.AdmissionWebhookRegister{
			Type: webhooks.MutatingType,
			Path: "/mutate-ibm-cs-pod",
			Hook: &admission.Webhook{
				Handler: &podpreset.Mutator{
					Client: mgr.GetClient(),
				},
			},
		},
		NsSelector: managedbyCSSelector,
	})
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
	}
	webhooks.Config.AddWebhook(webhooks.CSWebhook{
		Name:        "ibm-cs-ns-mapping-webhook-configuration",
		WebhookName: "cs-ns-mapping-configmap.operator.ibm.com",
		Rule: webhooks.NewRule().
			OneResource("", "v1", "configmaps").
			ForUpdate().
			ForCreate().
			NamespacedScope(),
		Register: webhooks.AdmissionWebhookRegister{
			Type: webhooks.MutatingType,
			Path: "/validate-ibm-cs-ns-map",
			Hook: &admission.Webhook{
				Handler: &podpreset.Mutator{
					Client: mgr.GetClient(),
				},
			},
		},
		NsSelector: v1.LabelSelector{
			MatchExpressions: []v1.LabelSelectorRequirement{
				{
					Key: "name",
					Operator: v1.LabelSelectorOpIn,
					Values: []string{
						"kube-public",
					},
				},
			},
		},
	})

	klog.Info("setting up webhook server")
	if err := webhooks.Config.SetupServer(mgr, namespace); err != nil {
		return err
	}

	return nil
}
