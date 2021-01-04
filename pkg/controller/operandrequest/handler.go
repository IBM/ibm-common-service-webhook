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

package operandrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	utilyaml "github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

// Mutator is the struct of webhook
// +k8s:deepcopy-gen=false
type Mutator struct {
	Reader  client.Reader
	decoder *admission.Decoder
}

type csMaps struct {
	NsMappingList []nsMapping `json:"namespaceMapping"`
	DefaultCsNs   string      `json:"defaultCsNs"`
}

type nsMapping struct {
	RequestNS string `json:"requested-from-namespace"`
	CsNs      string `json:"map-common-service-namespace"`
}

// Handle mutates every creating pods
func (p *Mutator) Handle(ctx context.Context, req admission.Request) admission.Response {

	klog.Infof("Webhook is invoked by OperandRequest %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)
	opreq := &odlmv1alpha1.OperandRequest{}
	ns := req.AdmissionRequest.Namespace
	err := p.decoder.Decode(req, opreq)
	if err != nil {
		klog.Error(err, "Error occurred decoding OperandRequest")
		return admission.Errored(http.StatusBadRequest, err)
	}
	copy := opreq.DeepCopy()

	err = p.mutatePodsFn(ctx, copy, ns)

	if err != nil {
		klog.Error(err, "Error occurred mutating OperandRequest")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaledOpreq, err := json.Marshal(opreq)
	marshaledcopy, err := json.Marshal(copy)

	// admission.PatchResponse generates a Response containing patches.
	return admission.PatchResponseFromRaw(marshaledOpreq, marshaledcopy)

}

// Mutates function values
func (p *Mutator) mutatePodsFn(ctx context.Context, opreq *odlmv1alpha1.OperandRequest, namespace string) error {

	csConfigmap := &corev1.ConfigMap{}

	err := p.Reader.Get(ctx, types.NamespacedName{Namespace: "kube-public", Name: "common-service-maps"}, csConfigmap)

	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("common service configmap kube-public/common-service-maps is not found: %v", err)
			return nil
		}
		return fmt.Errorf("failed to fetch configmap kube-public/common-service-maps: %v", err)
	}

	commonServiceMaps := csConfigmap.Data["common-service-maps.yaml"]
	var cmData csMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return err
	}

	var defaultCsNs string
	if cmData.DefaultCsNs == "" {
		defaultCsNs = "ibm-common-services"
	} else {
		defaultCsNs = cmData.DefaultCsNs
	}

	for _, nsMapping := range cmData.NsMappingList {
		if nsMapping.RequestNS == opreq.Namespace {
			for index, req := range opreq.Spec.Requests {
				if req.RegistryNamespace == defaultCsNs {
					req.RegistryNamespace = nsMapping.CsNs
					opreq.Spec.Requests[index] = req
				}
			}
			break
		}
	}

	return nil
}

// InjectDecoder injects the decoder into the Mutator
func (p *Mutator) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
