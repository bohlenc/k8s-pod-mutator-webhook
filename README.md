# k8s-pod-mutator-webhook
[![Webhook Image Build](https://img.shields.io/docker/cloud/automated/bohlenc/k8s-pod-mutator-webhook?label=webhook%20build)](https://hub.docker.com/r/bohlenc/k8s-pod-mutator-webhook) [![Webhook Image Version](https://img.shields.io/docker/v/bohlenc/k8s-pod-mutator-webhook?label=webhook&sort=semver)](https://hub.docker.com/r/bohlenc/k8s-pod-mutator-webhook) [![Init Image Build](https://img.shields.io/docker/cloud/automated/bohlenc/k8s-pod-mutator-init?label=init%20build)](https://hub.docker.com/r/bohlenc/k8s-pod-mutator-webhook) [![Init Image Version](https://img.shields.io/docker/v/bohlenc/k8s-pod-mutator-init?label=init&sort=semver)](https://hub.docker.com/r/bohlenc/k8s-pod-mutator-init)

This is a Kubernetes Mutating Admission Webhook (see https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).
It can apply arbitrary changes (a "patch") to a Pod's manifest. A patch can do anything from adding or changing metadata to containers and init-containers with volumes.

The Kubernetes API server only supports communication with webhooks over HTTPS - 
an init-container is included that automates cert generation and any necessary configuration (i.e. applying the `caBundle` to the `MutatingWebhookConfiguration`).

## Problem Statement

It is a recurring requirement in Kubernetes deployments to transparently mutate Pod manifests - 
either to add new functionality transparently to existing deployments and applications, or to enforce compliance and other policies and requirements. 

This webhook provides a flexible and scalable solution to those problems.


## Notable Options

### --patch
Path to the YAML file containing the patch to be applied to eligible Pods (see https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#pod-v1-core for help).

### --log-level
panic | fatal | error | warn | info | debug | trace


## Installation

### Helm
1. adjust the `values.yaml` in `deploy/helm` to your requirements
2. install the chart via `helm install k8s-pod-mutator deploy/helm -f deploy/helm/values.yaml`


By default, the webhook is reachable under "https://<service_name>:8443/mutate"


## Examples

Issue: https://github.com/Azure/azure-sdk-for-net/issues/18312

To apply the workaround proposed [here](https://github.com/Azure/azure-sdk-for-net/issues/18312#issuecomment-771116456)
simply install the Helm chart with the provided example values: 

`helm upgrade --install k8s-pod-mutator deploy/helm -f values.yaml -f examples/values.example.yaml`.

This example adds an init-container
```
spec:
  initContainers:
    - name: wait-for-imds
      image: busybox:1.33
      command: ['sh', '-c', 'wget "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/" --header "Metadata: true" -S --spider -T 6']
```
to all Pods that have a Label `aadpodidbinding`.


## Contributions

If you feel like anything is missing, should be fixed or could be improved, issues and pull requests are more than welcome.


## License

MIT
