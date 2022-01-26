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

package podpreset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv1alpha1 "github.com/IBM/ibm-common-service-webhook/pkg/apis/v1alpha1"
)

// Mutator is the struct of webhook
// +k8s:deepcopy-gen=false
type Mutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle mutates every creating pods
func (p *Mutator) Handle(ctx context.Context, req admission.Request) admission.Response {

	klog.Infof("Webhook is invoked by pod %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)
	pod := &corev1.Pod{}
	ns := req.AdmissionRequest.Namespace
	err := p.decoder.Decode(req, pod)
	if err != nil {
		klog.Error(err, "Error occurred decoding Pod")
		return admission.Errored(http.StatusBadRequest, err)
	}
	copy := pod.DeepCopy()

	err = p.mutatePodsFn(ctx, copy, ns)

	if err != nil {
		klog.Error(err, "Error occurred mutating Pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaledPod, err := json.Marshal(pod)
	marshaledcopy, err := json.Marshal(copy)

	// admission.PatchResponse generates a Response containing patches.
	return admission.PatchResponseFromRaw(marshaledPod, marshaledcopy)

}

// Mutates function values
func (p *Mutator) mutatePodsFn(ctx context.Context, pod *corev1.Pod, namespace string) error {

	if _, isMirrorPod := pod.Annotations[corev1.MirrorPodAnnotationKey]; isMirrorPod {
		return nil
	}

	// Ignore if exclusion annotation is present
	if podAnnotations := pod.GetAnnotations(); podAnnotations != nil {
		if podAnnotations[corev1.PodPresetOptOutAnnotationKey] == "true" {
			klog.Infof("Pod %s has been patched", pod.Name)
			return nil
		}
	}

	podPresetList := &operatorv1alpha1.PodPresetList{}

	err := p.Client.List(ctx, podPresetList, &client.ListOptions{})

	if err != nil {
		return fmt.Errorf("listing pod presets failed: %v", err)
	}

	matchingPPs, err := filterPodPresets(podPresetList, pod, namespace)
	if err != nil {
		return fmt.Errorf("filtering pod presets failed: %v", err)
	}

	if len(matchingPPs) == 0 {
		return nil
	}

	presetNames := make([]string, len(matchingPPs))
	for i, pp := range matchingPPs {
		presetNames[i] = pp.GetName()
	}

	// detect merge conflict
	err = safeToApplyPodPresetsOnPod(pod, matchingPPs)
	if err != nil {
		// conflict, ignore the error, but raise an event
		klog.Infof("conflict occurred while applying. Podpreset names: %s; Pod Name: %s", strings.Join(presetNames, ","), pod.GetGenerateName())
		return nil
	}

	applyPodPresetsOnPod(pod, matchingPPs)

	klog.Infof("applied podpresets. Podpreset names: %s; Pod Name: %s", strings.Join(presetNames, ","), pod.GetGenerateName())

	return nil
}

// applyPodPresetsOnPod updates the PodSpec with merged information from all the
// applicable PodPresets. It ignores the errors of merge functions because merge
// errors have already been checked in safeToApplyPodPresetsOnPod function.
func applyPodPresetsOnPod(pod *corev1.Pod, podPresets []*operatorv1alpha1.PodPreset) {
	if len(podPresets) == 0 {
		return
	}

	volumes, _ := mergeVolumes(pod.Spec.Volumes, podPresets)
	pod.Spec.Volumes = volumes

	if pod.Spec.DNSPolicy == corev1.DNSClusterFirst {
		if pod.Spec.DNSConfig == nil {
			pod.Spec.DNSConfig = &corev1.PodDNSConfig{}
		}
		if pod.Spec.DNSConfig.Options == nil {
			pod.Spec.DNSConfig.Options = []corev1.PodDNSConfigOption{}
		}
		exist := false
		for _, op := range pod.Spec.DNSConfig.Options {
			if (op == corev1.PodDNSConfigOption{Name: "single-request-reopen"}) {
				exist = true
			}
		}
		if !exist {
			pod.Spec.DNSConfig.Options = append(pod.Spec.DNSConfig.Options, corev1.PodDNSConfigOption{Name: "single-request-reopen"})
		}
	}

	for i, ctr := range pod.Spec.Containers {
		applyPodPresetsOnContainer(&ctr, podPresets)
		pod.Spec.Containers[i] = ctr
	}

	// add annotation
	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = map[string]string{}
	}

	for _, pp := range podPresets {
		pod.ObjectMeta.Annotations[fmt.Sprintf("%s/podpreset-%s", podpresetName, pp.GetName())] = pp.GetResourceVersion()
	}
}

// applyPodPresetsOnContainer injects envVars, VolumeMounts and envFrom from
// given podPresets in to the given container. It ignores conflict errors
// because it assumes those have been checked already by the caller.
func applyPodPresetsOnContainer(ctr *corev1.Container, podPresets []*operatorv1alpha1.PodPreset) {
	envVars, _ := mergeEnv(ctr.Env, podPresets)
	ctr.Env = envVars

	volumeMounts, _ := mergeVolumeMounts(ctr.VolumeMounts, podPresets)
	ctr.VolumeMounts = volumeMounts

	envFrom, _ := mergeEnvFrom(ctr.EnvFrom, podPresets)
	ctr.EnvFrom = envFrom
}

// filterPodPresets returns list of PodPresets which match given Pod.
func filterPodPresets(list *operatorv1alpha1.PodPresetList, pod *corev1.Pod, namespace string) ([]*operatorv1alpha1.PodPreset, error) {
	var matchingPPs []*operatorv1alpha1.PodPreset

	for _, pp := range list.Items {
		if pp.Namespace != namespace {
			continue
		}
		if &pp.Spec.Selector == nil {
			matchingPPs = append(matchingPPs, &pp)
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(&pp.Spec.Selector)
		if err != nil {
			return nil, fmt.Errorf("label selector conversion failed: %v for selector: %v", pp.Spec.Selector, err)
		}

		// check if the pod labels match the selector
		if !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		klog.Infof("PodPreset matches pod labels PodPreset: %s, Pod: %s", pp.GetName(), pod.GetGenerateName())
		matchingPPs = append(matchingPPs, &pp)
	}
	return matchingPPs, nil
}

// safeToApplyPodPresetsOnPod determines if there is any conflict in information
// injected by given PodPresets in the Pod.
func safeToApplyPodPresetsOnPod(pod *corev1.Pod, podPresets []*operatorv1alpha1.PodPreset) error {
	var errs []error

	// volumes attribute is defined at the Pod level, so determine if volumes
	// injection is causing any conflict.
	if _, err := mergeVolumes(pod.Spec.Volumes, podPresets); err != nil {
		errs = append(errs, err)
	}
	for _, ctr := range pod.Spec.Containers {
		if err := safeToApplyPodPresetsOnContainer(&ctr, podPresets); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// mergeVolumes merges given list of Volumes with the volumes injected by given
// podPresets. It returns an error if it detects any conflict during the merge.
func mergeVolumes(volumes []corev1.Volume, podPresets []*operatorv1alpha1.PodPreset) ([]corev1.Volume, error) {
	origVolumes := map[string]corev1.Volume{}
	for _, v := range volumes {
		origVolumes[v.Name] = v
	}

	mergedVolumes := make([]corev1.Volume, len(volumes))
	copy(mergedVolumes, volumes)

	var errs []error

	for _, pp := range podPresets {
		for _, v := range pp.Spec.Volumes {
			found, ok := origVolumes[v.Name]
			if !ok {
				// if we don't already have it append it and continue
				origVolumes[v.Name] = v
				mergedVolumes = append(mergedVolumes, v)
				continue
			}

			// make sure they are identical or throw an error
			if !reflect.DeepEqual(found, v) {
				errs = append(errs, fmt.Errorf("merging volumes for %s has a conflict on %s: \n%#v\ndoes not match\n%#v\n in container", pp.GetName(), v.Name, v, found))
			}
		}
	}

	err := utilerrors.NewAggregate(errs)
	if err != nil {
		return nil, err
	}

	if len(mergedVolumes) == 0 {
		return nil, nil
	}

	return mergedVolumes, err
}

// safeToApplyPodPresetsOnContainer determines if there is any conflict in
// information injected by given PodPresets in the given container.
func safeToApplyPodPresetsOnContainer(ctr *corev1.Container, podPresets []*operatorv1alpha1.PodPreset) error {
	var errs []error
	// check if it is safe to merge env vars and volume mounts from given podpresets and
	// container's existing env vars.
	if _, err := mergeEnv(ctr.Env, podPresets); err != nil {
		errs = append(errs, err)
	}
	if _, err := mergeVolumeMounts(ctr.VolumeMounts, podPresets); err != nil {
		errs = append(errs, err)
	}

	return utilerrors.NewAggregate(errs)
}

// mergeEnv merges a list of env vars with the env vars injected by given list podPresets.
// It returns an error if it detects any conflict during the merge.
func mergeEnv(envVars []corev1.EnvVar, podPresets []*operatorv1alpha1.PodPreset) ([]corev1.EnvVar, error) {
	origEnv := map[string]corev1.EnvVar{}
	for _, v := range envVars {
		origEnv[v.Name] = v
	}

	mergedEnv := make([]corev1.EnvVar, len(envVars))
	copy(mergedEnv, envVars)

	var errs []error

	for _, pp := range podPresets {
		for _, v := range pp.Spec.Env {

			found, ok := origEnv[v.Name]
			if !ok {
				// if we don't already have it append it and continue
				origEnv[v.Name] = v
				mergedEnv = append(mergedEnv, v)
				continue
			}

			// make sure they are identical or throw an error
			if !reflect.DeepEqual(found, v) {
				errs = append(errs, fmt.Errorf("merging env for %s has a conflict on %s: \n%#v\ndoes not match\n%#v\n in container", pp.GetName(), v.Name, v, found))
			}
		}
	}

	err := utilerrors.NewAggregate(errs)
	if err != nil {
		return nil, err
	}

	return mergedEnv, err
}

func mergeEnvFrom(envSources []corev1.EnvFromSource, podPresets []*operatorv1alpha1.PodPreset) ([]corev1.EnvFromSource, error) {
	var mergedEnvFrom []corev1.EnvFromSource

	mergedEnvFrom = append(mergedEnvFrom, envSources...)
	for _, pp := range podPresets {
		for _, envFromSource := range pp.Spec.EnvFrom {
			// internalEnvFrom := api.EnvFromSource{}
			// if err := apiscorev1.Convert_v1_EnvFromSource_To_core_EnvFromSource(&envFromSource, &internalEnvFrom, nil); err != nil {
			// 	return nil, err
			// }
			mergedEnvFrom = append(mergedEnvFrom, envFromSource)
		}

	}

	return mergedEnvFrom, nil
}

// mergeVolumeMounts merges given list of VolumeMounts with the volumeMounts
// injected by given podPresets. It returns an error if it detects any conflict during the merge.
func mergeVolumeMounts(volumeMounts []corev1.VolumeMount, podPresets []*operatorv1alpha1.PodPreset) ([]corev1.VolumeMount, error) {

	origVolumeMounts := map[string]corev1.VolumeMount{}
	volumeMountsByPath := map[string]corev1.VolumeMount{}
	for _, v := range volumeMounts {
		origVolumeMounts[v.Name] = v
		volumeMountsByPath[v.MountPath] = v
	}

	mergedVolumeMounts := make([]corev1.VolumeMount, len(volumeMounts))
	copy(mergedVolumeMounts, volumeMounts)

	var errs []error

	for _, pp := range podPresets {
		for _, v := range pp.Spec.VolumeMounts {
			found, ok := origVolumeMounts[v.Name]
			if !ok {
				// if we don't already have it append it and continue
				origVolumeMounts[v.Name] = v
				mergedVolumeMounts = append(mergedVolumeMounts, v)
			} else {
				// make sure they are identical or throw an error
				// shall we throw an error for identical volumeMounts ?
				if !reflect.DeepEqual(found, v) {
					errs = append(errs, fmt.Errorf("merging volume mounts for %s has a conflict on %s: \n%#v\ndoes not match\n%#v\n in container", pp.GetName(), v.Name, v, found))
				}
			}

			found, ok = volumeMountsByPath[v.MountPath]
			if !ok {
				// if we don't already have it append it and continue
				volumeMountsByPath[v.MountPath] = v
			} else {
				// make sure they are identical or throw an error
				if !reflect.DeepEqual(found, v) {
					errs = append(errs, fmt.Errorf("merging volume mounts for %s has a conflict on mount path %s: \n%#v\ndoes not match\n%#v\n in container", pp.GetName(), v.MountPath, v, found))
				}
			}
		}
	}

	err := utilerrors.NewAggregate(errs)
	if err != nil {
		return nil, err
	}

	return mergedVolumeMounts, err
}

// InjectDecoder injects the decoder into the Mutator
func (p *Mutator) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
