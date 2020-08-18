package webhooks

import (
	"context"
	"fmt"

	"k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// WebhookReconciler knows how to reconcile webhook configuration CRs
type WebhookReconciler interface {
	SetName(name string)
	SetWebhookName(webhookName string)
	SetRule(rule RuleWithOperations)
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

func (reconciler *CompositeWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	for _, innerReconciler := range reconciler.Reconcilers {
		if err := innerReconciler.Reconcile(ctx, client, caBundle); err != nil {
			return err
		}
	}

	return nil
}

type ValidatingWebhookReconciler struct {
	Path        string
	name        string
	webhookName string
	rule        RuleWithOperations
}

type MutatingWebhookReconciler struct {
	Path        string
	name        string
	webhookName string
	rule        RuleWithOperations
}

//Reconcile MutatingWebhookConfiguration
func (reconciler *MutatingWebhookReconciler) Reconcile(ctx context.Context, client k8sclient.Client, caBundle []byte) error {
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		ignorePolicy   = v1beta1.Ignore
		timeoutSeconds = int32(30)
	)

	cr := &v1beta1.MutatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s", reconciler.name),
		},
	}

	klog.Infof("Creating/Updating MutatingWebhook %s", fmt.Sprintf("%s", reconciler.name))
	_, err := controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.MutatingWebhook{
			{
				Name:        fmt.Sprintf("%s", reconciler.webhookName),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: "ibm-common-services",
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
		timeoutSeconds = int32(30)
	)

	cr := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s", reconciler.name),
		},
	}

	klog.Infof("Creating/Updating ValidatingWebhook %s", fmt.Sprintf("%s", reconciler.name))
	_, err := controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.ValidatingWebhook{
			{
				Name:        fmt.Sprintf("%s", reconciler.webhookName),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: "ibm-common-services",
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
