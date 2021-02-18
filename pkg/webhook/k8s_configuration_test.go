package webhook

import (
	"github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"testing"
)

func TestConfiguration_GetServiceMetadata(t *testing.T) {
	configuration := Configuration{
		template: &admissionregistrationv1.MutatingWebhookConfiguration{},
	}

	_ = yaml.Unmarshal([]byte(mutatingWebhookConfiguration), configuration.template)
	configuration.template.Webhooks[0].ClientConfig.Service = &admissionregistrationv1.ServiceReference{}
	_ = yaml.Unmarshal([]byte(mutatingWebhookConfiguration), configuration.template)

	metadata := configuration.GetServiceMetadata()
	assert.Equal(t, "RELEASE-NAME-k8s-pod-mutator-webhook", metadata.Name)
	assert.Equal(t, "RELEASE-NAMESPACE", metadata.Namespace)
}

const mutatingWebhookConfiguration = `
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: RELEASE-NAME-k8s-pod-mutator-webhook
  labels:
app.kubernetes.io/name: k8s-pod-mutator-webhook
app.kubernetes.io/instance: RELEASE-NAME
app.kubernetes.io/version: "1.0.0"
webhooks:
  - name: webhook.k8s-pod-mutator.io
    admissionReviewVersions: ["v1"]
    clientConfig:
      service:
        name: RELEASE-NAME-k8s-pod-mutator-webhook
        namespace: RELEASE-NAMESPACE
        path: "/mutate"
    rules:
      - operations: ["CREATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    matchPolicy: Equivalent
    sideEffects: None
    reinvocationPolicy: Never
    failurePolicy: Ignore
    timeoutSeconds: 2
    namespaceSelector:
      matchExpressions:
        - key: control-plane # ignore kube-system
          operator: DoesNotExist
`
