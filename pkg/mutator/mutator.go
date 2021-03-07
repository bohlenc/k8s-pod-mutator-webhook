package mutator

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s-pod-mutator-webhook/internal/admission_review"
	"k8s-pod-mutator-webhook/internal/logger"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

type MutationSettings struct {
	PatchFile string
}

type Mutator struct {
	patch *Patch
}

func CreateMutator(settings MutationSettings) (*Mutator, error) {
	logger.Logger.WithFields(logrus.Fields{
		"settings": fmt.Sprintf("%+v", settings),
	}).Infoln("creating mutator")

	patchYaml, err := ioutil.ReadFile(settings.PatchFile)
	if err != nil {
		return nil, fmt.Errorf("could not read patch file: %v", err)
	}

	patch, err := CreatePatch(patchYaml)
	if err != nil {
		return nil, err
	}

	return &Mutator{patch}, nil
}

func (m *Mutator) Mutate(request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		logger.Logger.WithFields(logrus.Fields{
			"error": err,
			"type":  reflect.TypeOf(pod),
		}).Errorln("unmarshalling failed")
		return admission_review.ErrorResponse(err)
	}

	podName := maybePodName(pod.ObjectMeta)
	ensurePodNamespace(request, &pod)

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

	jsonPatch, err := m.patch.Apply(&pod)
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

func ensurePodNamespace(request *admissionv1.AdmissionRequest, pod *corev1.Pod) {
	if pod.Namespace == "" {
		pod.Namespace = request.Namespace
	}
}
