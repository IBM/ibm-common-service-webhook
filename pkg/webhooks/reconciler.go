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

package webhooks

import (
	"context"
	"fmt"

	"k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/IBM/ibm-common-service-webhook/pkg/utils"
)

// WebhookReconciler knows how to reconcile webhook configuration CRs
type WebhookReconciler interface {
	SetName(name string)
	SetWebhookName(webhookName string)
	SetRule(rule RuleWithOperations)
	EnableNsSelector()
	Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error
}

type CompositeWebhookReconciler struct {
	Reconcilers []WebhookReconciler
}

func (reconciler *CompositeWebhookReconciler) SetName(name string) {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.SetName(name)
	}
}

func (reconciler *CompositeWebhookReconciler) SetWebhookName(webhookName string) {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.SetWebhookName(webhookName)
	}
}

func (reconciler *CompositeWebhookReconciler) SetRule(rule RuleWithOperations) {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.SetRule(rule)
	}
}

func (reconciler *CompositeWebhookReconciler) EnableNsSelector() {
	for _, innerReconciler := range reconciler.Reconcilers {
		innerReconciler.EnableNsSelector()
	}
}

func (reconciler *CompositeWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	for _, innerReconciler := range reconciler.Reconcilers {
		if err := innerReconciler.Reconcile(ctx, client, caBundle); err != nil {
			return err
		}
	}

	return nil
}

type ValidatingWebhookReconciler struct {
	Path             string
	name             string
	webhookName      string
	rule             RuleWithOperations
	enableNsSelector bool
}

type MutatingWebhookReconciler struct {
	Path             string
	name             string
	webhookName      string
	rule             RuleWithOperations
	enableNsSelector bool
}

//Reconcile MutatingWebhookConfiguration
func (reconciler *MutatingWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		ignorePolicy   = v1beta1.Ignore
		timeoutSeconds = int32(10)
	)

	namespace := utils.GetWatchNamespace()

	cr := &v1beta1.MutatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s", reconciler.name),
		},
	}

	webhookLabel := make(map[string]string)
	webhookLabel["managed-by-common-service-webhook"] = "true"

	klog.Infof("Creating/Updating MutatingWebhook %s", fmt.Sprintf("%s", reconciler.name))
	_, err := controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.MutatingWebhook{
			{
				Name:        fmt.Sprintf("%s", reconciler.webhookName),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: namespace,
						Name:      operatorPodServiceName,
						Path:      &reconciler.Path,
						Port:      &port,
					},
				},
				Rules: []v1beta1.RuleWithOperations{
					{
						Operations: reconciler.rule.Operations,
						Rule: v1beta1.Rule{
							APIGroups:   reconciler.rule.APIGroups,
							APIVersions: reconciler.rule.APIVersions,
							Resources:   reconciler.rule.Resources,
							Scope:       &reconciler.rule.Scope,
						},
					},
				},
				MatchPolicy:             &matchPolicy,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &ignorePolicy,
				TimeoutSeconds:          &timeoutSeconds,
			},
		}
		if reconciler.enableNsSelector {
			for index := range cr.Webhooks {
				cr.Webhooks[index].NamespaceSelector = &v1.LabelSelector{
					MatchLabels: webhookLabel,
				}
			}
		}
		return nil
	})
	if err != nil {
		klog.Error(err)
	}
	return err
}

//Reconcile ValidatingWebhookConfiguration
func (reconciler *ValidatingWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		failurePolicy  = v1beta1.Fail
		timeoutSeconds = int32(10)
	)

	namespace := utils.GetWatchNamespace()

	cr := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s", reconciler.name),
		},
	}

	webhookLabel := make(map[string]string)
	webhookLabel["managed-by-common-service-webhook"] = "true"

	klog.Infof("Creating/Updating ValidatingWebhook %s", fmt.Sprintf("%s", reconciler.name))
	_, err := controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.ValidatingWebhook{
			{
				Name:        fmt.Sprintf("%s", reconciler.webhookName),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: namespace,
						Name:      operatorPodServiceName,
						Path:      &reconciler.Path,
						Port:      &port,
					},
				},
				Rules: []v1beta1.RuleWithOperations{
					{
						Operations: reconciler.rule.Operations,
						Rule: v1beta1.Rule{
							APIGroups:   reconciler.rule.APIGroups,
							APIVersions: reconciler.rule.APIVersions,
							Resources:   reconciler.rule.Resources,
							Scope:       &reconciler.rule.Scope,
						},
					},
				},
				MatchPolicy:             &matchPolicy,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &timeoutSeconds,
			},
		}
		if reconciler.enableNsSelector {
			for index := range cr.Webhooks {
				cr.Webhooks[index].NamespaceSelector = &v1.LabelSelector{
					MatchLabels: webhookLabel,
				}
			}
		}
		return nil
	})
	if err != nil {
		klog.Error(err)
	}
	return err
}

func (reconciler *ValidatingWebhookReconciler) SetName(name string) {
	reconciler.name = name
}

func (reconciler *MutatingWebhookReconciler) SetName(name string) {
	reconciler.name = name
}

func (reconciler *ValidatingWebhookReconciler) SetWebhookName(webhookName string) {
	reconciler.webhookName = webhookName
}

func (reconciler *MutatingWebhookReconciler) SetWebhookName(webhookName string) {
	reconciler.webhookName = webhookName
}

func (reconciler *ValidatingWebhookReconciler) SetRule(rule RuleWithOperations) {
	reconciler.rule = rule
}

func (reconciler *MutatingWebhookReconciler) SetRule(rule RuleWithOperations) {
	reconciler.rule = rule
}

func (reconciler *MutatingWebhookReconciler) EnableNsSelector() {
	reconciler.enableNsSelector = true
}

func (reconciler *ValidatingWebhookReconciler) EnableNsSelector() {
	reconciler.enableNsSelector = true
}
