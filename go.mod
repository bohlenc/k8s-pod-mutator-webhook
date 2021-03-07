module k8s-pod-mutator-webhook

require (
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	gomodules.xyz/jsonpatch/v3 v3.0.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/yaml v1.2.0
)

go 1.15
