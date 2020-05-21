//
// Copyright 2020 IBM Corporation
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
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

const (
	webhookName             = "ibm-common-service-webhook"
	webhookSecretName       = "ibm-common-service-webhook-cert"
	serverPort        int32 = 8443
	certDir                 = "/tmp/cert"
	podpresetName           = "cs-podpreset.operator.ibm.com"
	webhookConfigName       = "ibm-common-service-webhook-configuration"
)

var log = logf.Log.WithName("controller_podpreset")

func Add(mgr manager.Manager) error {

	ns, err := k8sutil.GetWatchNamespace()

	if err != nil {
		return err
	}

	mutatingWebhook, err := builder.NewWebhookBuilder().
		Name(podpresetName).
		Mutating().
		Operations(admissionregistrationv1beta1.Create).
		WithManager(mgr).
		ForType(&corev1.Pod{}).
		FailurePolicy(admissionregistrationv1beta1.Ignore).
		Handlers(&Mutator{}).
		Build()

	if err != nil {
		log.Error(err, "Error occurred building mutating webhook")
		return err
	}

	svr, err := webhook.NewServer(webhookName, mgr, webhook.ServerOptions{
		Port:    serverPort,
		CertDir: certDir,
		BootstrapOptions: &webhook.BootstrapOptions{
			Secret: &types.NamespacedName{
				Namespace: ns,
				Name:      webhookSecretName,
			},
			Service: &webhook.Service{
				Namespace: ns,
				Name:      webhookName,
				Selectors: map[string]string{
					"name": webhookName,
				},
			},
			MutatingWebhookConfigName: webhookConfigName,
		},
	})

	if err != nil {
		log.Error(err, "Error occurred creating webhook server")
		return err
	}

	err = svr.Register(mutatingWebhook)

	if err != nil {
		log.Error(err, "Error occurred registrying webhook server")
		return err
	}

	return nil
}
