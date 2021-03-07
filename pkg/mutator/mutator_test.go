package mutator

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gomodules.xyz/jsonpatch/v3"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestMutator_MutateCanApplyChanges(t *testing.T) {
	testCases := []struct {
		pod               string
		patch             string
		expectedJsonPatch string
	}{
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			patch: `
spec:
  initContainers:
  - name: wait-for-imds
    image: busybox:1.33
    command: [
      "sh",
      "-c",
      "wget \"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/\" --header \"Metadata: true\" -S --spider -T 6"
    ]
`,
			expectedJsonPatch: `
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "k8s-pod-mutator.io/mutated": "true"
    }
  },
  {
    "op": "add",
    "path": "/spec/initContainers",
    "value": [
      {
        "command": [
          "sh",
          "-c",
          "wget \"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/\" --header \"Metadata: true\" -S --spider -T 6"
        ],
        "image": "busybox:1.33",
        "name": "wait-for-imds",
        "resources": {}
      }
    ]
  }
]
`,
		},
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			patch: `
spec:
  containers:
  - name: busybox
    image: busybox
`,
			expectedJsonPatch: `
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "k8s-pod-mutator.io/mutated": "true"
    }
  },
  {
    "op": "add",
    "path": "/spec/containers/1",
    "value": {
      "image": "alpine",
      "name": "alpine",
      "resources": {}
    }
  },
  {
    "op": "replace",
    "path": "/spec/containers/0/image",
    "value": "busybox"
  },
  {
    "op": "replace",
    "path": "/spec/containers/0/name",
    "value": "busybox"
  }
]
`,
		},
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			patch: `
spec:
  containers:
  - name: busybox
    image: busybox
    volumeMounts:
    - name: test
      mountPath: /tmp/test
  volumes:
  - name: test
    emptyDir: {}
`,
			expectedJsonPatch: `
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "k8s-pod-mutator.io/mutated": "true"
    }
  },
  {
    "op": "add",
    "path": "/spec/containers/1",
    "value": {
      "image": "alpine",
      "name": "alpine",
      "resources": {}
    }
  },
  {
    "op": "replace",
    "path": "/spec/containers/0/image",
    "value": "busybox"
  },
  {
    "op": "replace",
    "path": "/spec/containers/0/name",
    "value": "busybox"
  },
  {
    "op": "add",
    "path": "/spec/containers/0/volumeMounts",
    "value": [
      {
        "mountPath": "/tmp/test",
        "name": "test"
      }
    ]
  },
  {
    "op": "add",
    "path": "/spec/volumes",
    "value": [
      {
        "emptyDir": {},
        "name": "test"
      }
    ]
  }
]
`,
		},
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "container1",
		"image": "alpine"
	  },
	  {
		"name": "container2",
		"image": "alpine"
	  }
	]
  }
}`,
			patch: `
spec:
  initContainers:
  - name: ca
    image: alpine
    command: ["sh", "-c", "cp -r /etc/ssl/certs /volume"]
    volumeMounts:
    - mountPath: /volume
      name: cacerts
  containers:
  - name: "*"
    volumeMounts:
    - mountPath: /etc/ssl
      name: cacerts
  volumes:
  - name: cacerts
    emptyDir: {}
`,
			expectedJsonPatch: `
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "k8s-pod-mutator.io/mutated": "true"
    }
  },
  {
    "op": "add",
    "path": "/spec/containers/0/volumeMounts",
    "value": [
      {
        "mountPath": "/etc/ssl",
        "name": "cacerts"
      }
    ]
  },
  {
    "op": "add",
    "path": "/spec/containers/1/volumeMounts",
    "value": [
      {
        "mountPath": "/etc/ssl",
        "name": "cacerts"
      }
    ]
  },
  {
    "op": "add",
    "path": "/spec/initContainers",
    "value": [
      {
        "command": [
          "sh",
          "-c",
          "cp -r /etc/ssl/certs /volume"
        ],
        "image": "alpine",
        "name": "ca",
        "resources": {},
        "volumeMounts": [
          {
            "mountPath": "/volume",
            "name": "cacerts"
          }
        ]
      }
    ]
  },
  {
    "op": "add",
    "path": "/spec/volumes",
    "value": [
      {
        "emptyDir": {},
        "name": "cacerts"
      }
    ]
  }
]
`,
		},
	}

	for testNo, testCase := range testCases {
		admissionRequest := v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(testCase.pod),
			},
		}
		mutator := &Mutator{
			patch: createPatch(testCase.patch),
		}

		admissionResponse := mutator.Mutate(&admissionRequest)
		println(fmt.Sprintf("TestNo: %v", testNo))

		expected := unmarshalJsonPatch([]byte(testCase.expectedJsonPatch))
		actual := unmarshalJsonPatch(admissionResponse.Patch)
		assert.ElementsMatch(t, expected, actual)
	}
}

func TestMutator_MutateConsidersStatusAnnotationForEligibility(t *testing.T) {
	testCases := []struct {
		pod                  string
		expectHasBeenPatched bool
	}{
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod",
	"annotations": {
		"k8s-pod-mutator.io/mutated": "true"
	}
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			expectHasBeenPatched: false,
		},
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			expectHasBeenPatched: true,
		},
		{
			pod: `
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod",
	"annotations": {
		"k8s-pod-mutator.io/mutated": "false"
	}
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			expectHasBeenPatched: true,
		},
	}

	for _, testCase := range testCases {
		admissionRequest := v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(testCase.pod),
			},
		}
		mutator := &Mutator{
			patch: createPatch(`
metadata:
  labels:
    added-label: test
`,
			),
		}

		admissionResponse := mutator.Mutate(&admissionRequest)
		assert.True(t, admissionResponse.Allowed)
		if testCase.expectHasBeenPatched {
			assert.NotNil(t, admissionResponse.Patch)
			assert.NotNil(t, admissionResponse.PatchType)
		} else {
			assert.Nil(t, admissionResponse.Patch)
			assert.Nil(t, admissionResponse.PatchType)
		}
	}

}

func TestMutator_MutateSetsStatusAnnotationTrue(t *testing.T) {
	admissionRequest := v1.AdmissionRequest{
		Object: runtime.RawExtension{
			Raw: []byte(`
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
	"name": "test-pod"
  },
  "spec": {
	"containers": [
	  {
		"name": "alpine",
		"image": "alpine"
	  }
	]
  }
}`,
			),
		},
	}

	mutator := &Mutator{
		patch: createPatch(`
metadata:
  labels:
    added-label: test
`,
		),
	}

	admissionResponse := mutator.Mutate(&admissionRequest)

	expected := unmarshalJsonPatch([]byte(`
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "k8s-pod-mutator.io/mutated": "true"
    }
  },
  {
    "op": "add",
    "path": "/metadata/labels",
    "value": {
      "added-label": "test"
    }
  }
]
`))
	actual := unmarshalJsonPatch(admissionResponse.Patch)

	assert.Equal(t, expected, actual)
}

func unmarshalJsonPatch(patchBytes []byte) []jsonpatch.Operation {
	var patch []jsonpatch.Operation
	err := json.Unmarshal(patchBytes, &patch)
	if err != nil {
		panic(err)
	}
	return patch
}

func createPatch(patchYaml string) *Patch {
	patch, err := CreatePatch([]byte(patchYaml))
	if err != nil {
		panic(err)
	}
	return patch
}
