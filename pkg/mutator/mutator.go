package mutator

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gomodules.xyz/jsonpatch/v3"
	"io/ioutil"
	"k8s-pod-mutator-webhook/internal/admission_review"
	"k8s-pod-mutator-webhook/internal/logger"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"reflect"
)

const statusAnnotation = "k8s-pod-mutator.io/mutated"

type MutationSettings struct {
	PatchFile string
}

type Mutator struct {
	patchJsonBytes []byte
}

func CreateMutator(settings MutationSettings) (*Mutator, error) {
	logger.Logger.WithFields(logrus.Fields{
		"settings": fmt.Sprintf("%+v", settings),
	}).Infoln("creating mutator")

	patchJsonBytes, err := readAsJsonBytes(settings.PatchFile)
	if err != nil {
		return nil, err
	}

	return &Mutator{patchJsonBytes}, nil
}

func readAsJsonBytes(patchFile string) ([]byte, error) {
	logger.Logger.WithFields(logrus.Fields{
		"patchFile": patchFile,
	}).Tracef("reading patch file...")
	patchYamlBytes, err := ioutil.ReadFile(patchFile)
	if err != nil {
		return nil, fmt.Errorf("could not read patch file: %v", err)
	}
	logger.Logger.Debugf("patch yaml: %v", string(patchYamlBytes))

	patchJsonBytes, err := yaml.ToJSON(patchYamlBytes)
	if err != nil {
		return nil, fmt.Errorf("could not convert patch from yaml to json: %v", err)
	}
	logger.Logger.Tracef("patch json: %v", string(patchJsonBytes))
	return patchJsonBytes, nil
}

func (i *Mutator) Mutate(request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		logger.Logger.WithFields(logrus.Fields{
			"error": err,
			"type":  reflect.TypeOf(pod),
		}).Errorln("decode failed")
		return admission_review.ErrorResponse(err)
	}

	podName := maybePodName(pod.ObjectMeta)
	ensurePodNamespace(request, pod)

	logger.Logger.WithFields(logrus.Fields{
		"namespace": pod.Namespace,
		"name":      podName,
	}).Infoln("mutation requested")
	logger.Logger.Tracef("Object.Raw: %v", string(request.Object.Raw))

	if alreadyMutated(&pod) {
		logger.Logger.WithFields(logrus.Fields{
			"namespace": podName,
			"name":      pod.Namespace,
			"reason":    "already mutated",
		}).Infoln("mutation skipped")
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	jsonPatch, err := createJsonPatch(&pod, i.patchJsonBytes)
	if err != nil {
		logger.Logger.Errorf("could not create json patch: %v", err)
		return admission_review.ErrorResponse(err)
	}

	response := &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   jsonPatch,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	logger.Logger.WithFields(logrus.Fields{
		"namespace": pod.Namespace,
		"name":      podName,
	}).Infoln("mutation succeeded")

	return response
}

func alreadyMutated(pod *corev1.Pod) bool {
	return pod.Annotations[statusAnnotation] == "true"
}

func maybePodName(metadata metav1.ObjectMeta) string {
	if metadata.Name != "" {
		return metadata.Name
	}
	if metadata.GenerateName != "" {
		return metadata.GenerateName + "***** (actual name not yet known)"
	}
	return ""
}

func ensurePodNamespace(request *admissionv1.AdmissionRequest, pod corev1.Pod) {
	if pod.Namespace == "" {
		pod.Namespace = request.Namespace
	}
}

func createJsonPatch(pod *corev1.Pod, patchJson []byte) ([]byte, error) {
	originalJson, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("could not encode pod: %v", err)
	}
	logger.Logger.Tracef("originalJson: %v", string(originalJson))

	overlayedJson, err := strategicpatch.StrategicMergePatch(originalJson, patchJson, corev1.Pod{})
	if err != nil {
		return nil, fmt.Errorf("could not apply strategic merge patch: %v", err)
	}

	overlayedJson, err = markMutated(overlayedJson)
	if err != nil {
		return nil, fmt.Errorf("could not set status annotation: %v", err)
	}
	logger.Logger.Tracef("overlayedJson: %v", string(overlayedJson))

	jsonPatch, err := jsonpatch.CreatePatch(originalJson, overlayedJson)
	if err != nil {
		return nil, fmt.Errorf("could not create two-way merge patch: %v", err)
	}
	logger.Logger.Tracef("jsonPatch: %v", jsonPatch)

	return json.Marshal(jsonPatch)
}

func markMutated(overlayedJson []byte) ([]byte, error) {
	overlayedPod := &corev1.Pod{}
	if err := json.Unmarshal(overlayedJson, overlayedPod); err != nil {
		return nil, err
	}

	if len(overlayedPod.Annotations) == 0 {
		overlayedPod.Annotations = make(map[string]string)
	}

	overlayedPod.Annotations[statusAnnotation] = "true"

	return json.Marshal(overlayedPod)
}
