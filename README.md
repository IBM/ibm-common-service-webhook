# ibm-common-service-webhook

It is a pod preset mutating admission webhook for the ibm-common-service-operator. Check the design document [here](./docs/ibm-common-service-webhook.md).
This operator is from implement of [podpreset](https://kubernetes.io/docs/concepts/workloads/pods/podpreset/)

In order to solve the a known [dns issue](https://github.com/kubernetes/kubernetes/issues/56903) which causes a 5 seconds dns resolving delay in the Openshift and Kubernetes.

This webhook will add dnsconfig into the pods

```yaml
template:
  spec:
    dnsConfig:
      options:
        - name: single-request-reopen
```

## Supported platforms

 - Red Hat OpenShift Container Platform 4.2 or newer installed on one of the following platforms:

   - Linux x86_64
   - Linux on Power (ppc64le)
   - Linux on IBM Z and LinuxONE


## Operator versions

The operator version is 0.0.1

### Quick start guide

Use the following quick start commands for building and testing the operator:
```
make install
```

### Debugging guide

Use the following commands to debug the operator:

```
oc describe deploy <ibm-common-service-webhook pod> -n ibm-common-services
oc logs <ibm-common-service-webhook pod>
oc get MutatingWebhookConfiguration mutating-webhook-configuration -oyaml
```
