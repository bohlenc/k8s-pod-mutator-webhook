package cert_generator

import (
	"github.com/stretchr/testify/assert"
	"k8s-pod-mutator-webhook/pkg/webhook"
	"testing"
)

func TestGenerate(t *testing.T) {
	_, err := Generate(
		webhook.ServiceMetadata{
			Name:      "some-service-name",
			Namespace: "some-service-namespace",
		},
		CertOutputFiles{
			CaCertOutputFile:  "/tmp/ca.crt",
			CaKeyOutputFile:   "/tmp/ca.key",
			TlsCertOutputFile: "/tmp/tls.crt",
			TlsKeyOutputFile:  "/tmp/tls.key",
		},
	)
	assert.Nil(t, err)
}
