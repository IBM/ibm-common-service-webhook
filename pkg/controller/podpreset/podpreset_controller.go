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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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

	// Setup webhooks
	log.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	log.Info("registering webhooks to the webhook server")
	hookServer.Register("/mutate-ibm-cs-pod", &webhook.Admission{Handler: &Mutator{}})

	return nil
}
