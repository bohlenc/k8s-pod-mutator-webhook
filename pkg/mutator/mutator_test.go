package mutator

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

// TODO: readability of expectations
func TestMutator_MutateCanApplyChanges(t *testing.T) {
	testCases := []struct {
		objectRaw string
		patch     string
		expected  string
	}{
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patch: `
				{
				  "spec": {
					"initContainers": [
					  {
						"name": "wait-for-imds",
						"image": "busybox:1.33",
						"command": [
						  "sh",
						  "-c",
						  "wget \"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/\" --header \"Metadata: true\" -S --spider -T 6"
						]
					  }
					]
				  }
				}`,
			expected: "[{\"op\":\"add\",\"path\":\"/metadata/annotations\",\"value\":{\"k8s-pod-mutator.io/mutated\":\"true\"}},{\"op\":\"add\",\"path\":\"/spec/initContainers\",\"value\":[{\"name\":\"wait-for-imds\",\"image\":\"busybox:1.33\",\"command\":[\"sh\",\"-c\",\"wget \\\"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01\\u0026resource=https://management.azure.com/\\\" --header \\\"Metadata: true\\\" -S --spider -T 6\"],\"resources\":{}}]}]",
		},
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patch: `
				{
				  "spec": {
					"containers": [
					  {
						"name": "busybox",
						"image": "busybox"
					  }
					]
				  }
				}`,
			expected: "[{\"op\":\"add\",\"path\":\"/metadata/annotations\",\"value\":{\"k8s-pod-mutator.io/mutated\":\"true\"}},{\"op\":\"add\",\"path\":\"/spec/containers/1\",\"value\":{\"name\":\"alpine\",\"image\":\"alpine\",\"resources\":{}}},{\"op\":\"replace\",\"path\":\"/spec/containers/0/name\",\"value\":\"busybox\"},{\"op\":\"replace\",\"path\":\"/spec/containers/0/image\",\"value\":\"busybox\"}]",
		},
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patch: `
				{
				  "spec": {
					"containers": [
					  {
						"name": "busybox",
						"image": "busybox",
						"volumeMounts": [
						  {
							"name": "test",
							"mountPath": "/tmp/test"
						  }
						]
					  }
					],
					"volumes": [
					  {
						"name": "test",
						"emptyDir": {}
					  }
					]
				  }
				}`,
			expected: "[{\"op\":\"add\",\"path\":\"/metadata/annotations\",\"value\":{\"k8s-pod-mutator.io/mutated\":\"true\"}},{\"op\":\"add\",\"path\":\"/spec/volumes\",\"value\":[{\"name\":\"test\",\"emptyDir\":{}}]},{\"op\":\"add\",\"path\":\"/spec/containers/1\",\"value\":{\"name\":\"alpine\",\"image\":\"alpine\",\"resources\":{}}},{\"op\":\"replace\",\"path\":\"/spec/containers/0/name\",\"value\":\"busybox\"},{\"op\":\"replace\",\"path\":\"/spec/containers/0/image\",\"value\":\"busybox\"},{\"op\":\"add\",\"path\":\"/spec/containers/0/volumeMounts\",\"value\":[{\"name\":\"test\",\"mountPath\":\"/tmp/test\"}]}]",
		},
	}

	for _, testCase := range testCases {
		admissionRequest := v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(testCase.objectRaw),
			},
		}
		mutator := &Mutator{
			patchJsonBytes: []byte(testCase.patch),
		}

		admissionResponse := mutator.Mutate(&admissionRequest)
		assert.Equal(t, testCase.expected, string(admissionResponse.Patch))
	}
}

func TestMutator_MutateFailsOnError(t *testing.T) {
	testCases := []struct {
		objectRaw string
		patch     string
	}{
		{
			objectRaw: "{ invalid pod }",
			patch: `
				{
				  "spec": {
					"containers": [
					  {
						"name": "busybox",
						"image": "busybox"
					  }
					]
				  }
				}`,
		},
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patch: "{ invalid patch }",
		},
	}

	for _, testCase := range testCases {
		admissionRequest := v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(testCase.objectRaw),
			},
		}
		mutator := &Mutator{
			patchJsonBytes: []byte(testCase.patch),
		}

		admissionResponse := mutator.Mutate(&admissionRequest)
		assert.Equal(t, int32(500), admissionResponse.Result.Code)
		assert.Equal(t, "Failure", admissionResponse.Result.Status)
		assert.NotNil(t, admissionResponse.Result.Message)
	}
}

func TestMutator_MutateConsidersStatusAnnotationForEligibility(t *testing.T) {
	testCases := []struct {
		objectRaw     string
		patchExpected bool
	}{
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod",
					"annotations": {
						"k8s-pod-mutator.io/mutated":"true"
					}
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patchExpected: false,
		},
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patchExpected: true,
		},
		{
			objectRaw: `
				{
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod",
					"annotations": {
						"k8s-pod-mutator.io/mutated":"false"
					}
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`,
			patchExpected: true,
		},
	}

	for _, testCase := range testCases {
		admissionRequest := v1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(testCase.objectRaw),
			},
		}
		mutator := &Mutator{
			patchJsonBytes: []byte(`
				{
				  "metadata":{
					"labels":{
					  "some-key":"some-value"
					}
				  }
				}`),
		}

		admissionResponse := mutator.Mutate(&admissionRequest)
		assert.True(t, admissionResponse.Allowed)
		if testCase.patchExpected {
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
				  "apiVersion":"v1",
				  "kind":"Pod",
				  "metadata":{
					"name":"test-pod"
				  },
				  "spec":{
					"containers":[
					  {
						"name":"alpine",
						"image":"alpine"
					  }
					]
				  }
				}`),
		},
	}

	mutator := &Mutator{
		patchJsonBytes: []byte(`
			{
			  "metadata":{
				"labels":{
				  "added-label":"test"
				}
			  }
			}`),
	}

	admissionResponse := mutator.Mutate(&admissionRequest)
	assert.Equal(t,
		"[{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{\"added-label\":\"test\"}},{\"op\":\"add\",\"path\":\"/metadata/annotations\",\"value\":{\"k8s-pod-mutator.io/mutated\":\"true\"}}]",
		string(admissionResponse.Patch),
	)
}
