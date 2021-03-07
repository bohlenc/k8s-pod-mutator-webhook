package mutator

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gomodules.xyz/jsonpatch/v3"
	"k8s-pod-mutator-webhook/internal/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
	"sync"
)

const statusAnnotation = "k8s-pod-mutator.io/mutated"

type Patch struct {
	mutex     sync.Mutex
	template  *corev1.Pod
	wildcards Wildcards
}

type Wildcards struct {
	initContainer *corev1.Container
	container     *corev1.Container
	volume        *corev1.Volume
}

func CreatePatch(patchYaml []byte) (*Patch, error) {
	logger.Logger.WithFields(logrus.Fields{
		"patchYaml": string(patchYaml),
	}).Infoln("creating patch")

	patch := &corev1.Pod{}
	err := yaml.Unmarshal(patchYaml, patch)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal patch from yaml: %v", err)
	}

	patch, wildcards, err := splitWildcards(patch)
	if err != nil {
		return nil, err
	}

	if len(patch.Annotations) == 0 {
		patch.Annotations = make(map[string]string)
	}
	patch.Annotations[statusAnnotation] = "true"

	return &Patch{
		template:  patch,
		wildcards: *wildcards,
	}, nil
}

func splitWildcards(patch *corev1.Pod) (*corev1.Pod, *Wildcards, error) {
	logger.Logger.WithFields(logrus.Fields{
		"patch": patch,
	}).Tracef("splitting wildcards")

	wildcardInitContainer, initContainers, err := splitContainers(patch.Spec.InitContainers)
	if err != nil {
		return nil, nil, err
	}

	wildcardContainer, containers, err := splitContainers(patch.Spec.Containers)
	if err != nil {
		return nil, nil, err
	}

	wildcardVolume, volumes, err := splitVolumes(patch.Spec.Volumes)
	if err != nil {
		return nil, nil, err
	}

	patch.Spec.InitContainers = initContainers
	patch.Spec.Containers = containers
	patch.Spec.Volumes = volumes

	wildcards := &Wildcards{
		initContainer: wildcardInitContainer,
		container:     wildcardContainer,
		volume:        wildcardVolume,
	}

	return patch, wildcards, nil
}

func splitContainers(allContainers []corev1.Container) (*corev1.Container, []corev1.Container, error) {
	var wildcard *corev1.Container
	var containers []corev1.Container
	for _, container := range allContainers {
		if container.Name == "*" {
			if wildcard != nil {
				return nil, nil, fmt.Errorf("only one wildcard is supported")
			}
			wildcard = &container
		} else {
			containers = append(containers, container)
		}
	}
	logger.Logger.WithFields(logrus.Fields{
		"wildcard":   wildcard,
		"containers": containers,
	}).Debugln("split containers")
	return wildcard, containers, nil
}

func splitVolumes(allVolumes []corev1.Volume) (*corev1.Volume, []corev1.Volume, error) {
	var wildcard *corev1.Volume
	var volumes []corev1.Volume
	for _, volume := range allVolumes {
		if volume.Name == "*" {
			if wildcard != nil {
				return nil, nil, fmt.Errorf("only one wildcard is supported")
			}
			wildcard = &volume
		} else {
			volumes = append(volumes, volume)
		}
	}
	logger.Logger.WithFields(logrus.Fields{
		"wildcard": wildcard,
		"volumes":  volumes,
	}).Debugln("split volumes")
	return wildcard, volumes, nil
}

func (p *Patch) Apply(pod *corev1.Pod) ([]byte, error) {
	p.mutex.Lock()

	p.appendApplicableWildcards(pod)

	patchJson, err := json.Marshal(p.template)
	if err != nil {
		return nil, fmt.Errorf("could not marshal patch to json: %v", err)
	}
	logger.Logger.Tracef("patchJson: %v", string(patchJson))

	podJson, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("could not marshal pod to json: %v", err)
	}
	logger.Logger.Tracef("podJson: %v", string(podJson))

	overlayedJson, err := strategicpatch.StrategicMergePatch(podJson, patchJson, corev1.Pod{})
	if err != nil {
		return nil, fmt.Errorf("could not apply strategic merge patch: %v", err)
	}
	logger.Logger.Tracef("overlayedJson: %v", string(overlayedJson))

	jsonPatch, err := jsonpatch.CreatePatch(podJson, overlayedJson)
	if err != nil {
		return nil, fmt.Errorf("could not create jsonpatch: %v", err)
	}

	jsonPatch = postProcess(jsonPatch)

	for i, operation := range jsonPatch {
		if operation.Path == "/metadata/creationTimestamp" && operation.Operation == "remove" ||
			(operation.Path == "/spec/containers" && operation.Operation == "remove") {
			jsonPatch[i] = jsonPatch[len(jsonPatch)-1]
			jsonPatch = jsonPatch[:len(jsonPatch)-1]
		}
	}

	logger.Logger.Tracef("jsonPatch: %v", jsonPatch)

	p.mutex.Unlock()

	return json.Marshal(jsonPatch)
}

func postProcess(original []jsonpatch.Operation) []jsonpatch.Operation {
	// workaround, or else patch contains unwanted operations resulting from unmarshalling JSON to corev1.Pod
	var processed []jsonpatch.Operation
	for _, operation := range original {
		if operation.Path == "/metadata/creationTimestamp" && operation.Operation == "remove" {
			continue
		}
		if operation.Path == "/spec/containers" && operation.Operation == "remove" {
			continue
		}
		processed = append(processed, operation)
	}
	return processed
}

func (p *Patch) appendApplicableWildcards(pod *corev1.Pod) {
	if p.wildcards.initContainer != nil {
		p.template.Spec.InitContainers = appendContainerWildcards(pod.Spec.InitContainers, p.template.Spec.InitContainers, *p.wildcards.initContainer)
	}
	if p.wildcards.container != nil {
		p.template.Spec.Containers = appendContainerWildcards(pod.Spec.Containers, p.template.Spec.Containers, *p.wildcards.container)
	}
	if p.wildcards.volume != nil {
		p.template.Spec.Volumes = appendVolumeWildcards(pod.Spec.Volumes, p.template.Spec.Volumes, *p.wildcards.volume)
	}
}

func appendContainerWildcards(podContainers []corev1.Container, patchContainers []corev1.Container, wildcard corev1.Container) []corev1.Container {
	for _, podContainer := range podContainers {
		wildcard.Name = podContainer.Name
		patchContainers = append(patchContainers, wildcard)
	}
	return patchContainers
}

func appendVolumeWildcards(podVolumes []corev1.Volume, patchVolumes []corev1.Volume, wildcard corev1.Volume) []corev1.Volume {
	for _, podVolume := range podVolumes {
		wildcard.Name = podVolume.Name
		patchVolumes = append(patchVolumes, wildcard)
	}
	return patchVolumes
}
