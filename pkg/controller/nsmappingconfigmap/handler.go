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

package nsmappingconfigmap

import (
	"context"
	"fmt"
	"net/http"

	utilyaml "github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Mutator is the struct of webhook
// +k8s:deepcopy-gen=false
type Mutator struct {
	Reader  client.Reader
	decoder *admission.Decoder
}

type csMaps struct {
	ControlNs     string      `json:"controlNamespace"`
	NsMappingList []nsMapping `json:"namespaceMapping"`
}

type nsMapping struct {
	RequestNS []string `json:"requested-from-namespace"`
	CsNs      string   `json:"map-to-common-service-namespace"`
}

// Handle mutates every creating pods
func (p *Mutator) Handle(ctx context.Context, req admission.Request) admission.Response {

	if req.Name != "common-service-maps" || req.Namespace != "kube-public" {
		return admission.Allowed("")
	}

	klog.Infof("Webhook is invoked by Configmap %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)
	cm := &corev1.ConfigMap{}
	err := p.decoder.Decode(req, cm)
	if err != nil {
		klog.Error(err, "Error occurred decoding OperandRequest")
		return admission.Errored(http.StatusBadRequest, err)
	}

	commonServiceMaps := cm.Data["common-service-maps.yaml"]
	var cmData csMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	CsNsSet := make(map[string]interface{})
	RequestNsSet := make(map[string]interface{})

	for _, nsMapping := range cmData.NsMappingList {
		// validate masterNamespace and controlNamespace
		if cmData.ControlNs == nsMapping.CsNs {
			return admission.Denied(fmt.Sprintf("controlNamespace: %v cannot be the same as one of the map-to-common-service-namespace", cmData.ControlNs))
		}
		if _, ok := CsNsSet[nsMapping.CsNs]; ok {
			return admission.Denied(fmt.Sprintf("map-to-common-service-namespace: %v exists in other namespace mappings", nsMapping.CsNs))
		}
		CsNsSet[nsMapping.CsNs] = struct{}{}
		// validate CloudPak Namespace and controlNamespace
		for _, ns := range nsMapping.RequestNS {
			if cmData.ControlNs == ns {
				return admission.Denied(fmt.Sprintf("controlNamespace: %v cannot be the same as one of the requested-from-namespace", cmData.ControlNs))
			}
			if _, ok := RequestNsSet[ns]; ok {
				return admission.Denied(fmt.Sprintf("There are multiple %v exit in the requested-from-namespace", ns))
			}
			RequestNsSet[ns] = struct{}{}
		}
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.Allowed("")

}

// InjectDecoder injects the decoder into the Validator
func (p *Mutator) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
