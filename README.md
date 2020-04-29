# ibm-common-service-webhook
It is a pod preset mutating admission webhook for the ibm-common-service-operator

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
