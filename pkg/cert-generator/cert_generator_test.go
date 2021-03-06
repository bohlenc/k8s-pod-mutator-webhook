package cert_generator

import (
	"github.com/stretchr/testify/assert"
	"k8s-pod-mutator-webhook/pkg/webhook"
	"testing"
)

func TestGenerate(t *testing.T) {
	certs, _ := Generate(
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

	assert.NotNil(t, certs.CaCert)
	assert.NotNil(t, certs.CaKey)
	assert.NotNil(t, certs.TlsCert)
	assert.NotNil(t, certs.TlsKey)
}
