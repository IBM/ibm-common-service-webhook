# ibm-common-service-webhook

The ibm common service webhook is used to inject other runtime requirements into the Pod at creation time. Mutating admission webhooks are invoked first, and can modify objects sent to the API server to enforce custom defaults. A CustomResourceDefinition (CRD) called PodPreset in the `operator.ibm.com` API group has the same specifications as [upstream API resources](https://kubernetes.io/docs/concepts/workloads/pods/podpreset/).

## Overview

This operator will install `single-request-reopen` into the dnsConfig of the pods and also it can support propreset functions from upstream. You can take a look at [How to use podpreset](https://kubernetes.io/docs/tasks/inject-data-application/podpreset/) for more information.

The following is an example of a PodPreset that injects to pods with the label `app: nginx` in the `nginx-namespace` namespce.

```yaml
apiVersion: operator.ibm.com/v1alpha1
kind: PodPreset
metadata:
  name: nginx-patch
  namespce: nginx-namespace
spec:
  selector:
    matchLabels:
      app: nginx
```

## Install

The ibm common service webhook will be installed by [ibm common service operator](https://github.com/IBM/ibm-common-service-operator) in 2Q 2020.

By default the following PodPreset is installed

```yaml
apiVersion: operator.ibm.com/v1alpha1
kind: PodPreset
metadata:
  name: ibm-common-service-webhook
  namespace: ibm-common-services
spec: {}
```

The webhook will insert all the pods in the `ibm-common-service` namespace.

### How Cloud Paks use the webhook

If Cloud Paks want to use this webhook to solve the [dns issue](https://github.com/kubernetes/kubernetes/issues/56903).
Cloud Paks can create the PodPreset is the Cloud Pak namespaces.

```yaml
apiVersion: operator.ibm.com/v1alpha1
kind: PodPreset
metadata:
  name: ibm-common-service-webhook
  namespace: ibm-cloud-paks
spec: {}
```

Then all the pods in the `ibm-cloud-paks` namespace will be inserted by the webhook.
