package controllers

import (
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

// PodScheduling holds the operator-wide placement constraints for game pods.
// Values set on an App's pod template take precedence.
type PodScheduling struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
}

// ParsePodScheduling accepts JSON or YAML. An empty string yields no constraints.
func ParsePodScheduling(raw string) (PodScheduling, error) {
	var scheduling PodScheduling
	if raw == "" {
		return scheduling, nil
	}
	if err := sigsyaml.UnmarshalStrict([]byte(raw), &scheduling); err != nil {
		return PodScheduling{}, fmt.Errorf("invalid game pod scheduling config: %w", err)
	}
	if scheduling.Affinity != nil && *scheduling.Affinity == (corev1.Affinity{}) {
		scheduling.Affinity = nil
	}
	return scheduling, nil
}

func (s PodScheduling) ApplyTo(spec *corev1.PodSpec) {
	if len(s.NodeSelector) > 0 {
		nodeSelector := maps.Clone(s.NodeSelector)
		maps.Copy(nodeSelector, spec.NodeSelector)
		spec.NodeSelector = nodeSelector
	}

	for _, toleration := range s.Tolerations {
		if !hasToleration(spec.Tolerations, toleration) {
			spec.Tolerations = append(spec.Tolerations, toleration)
		}
	}

	if spec.Affinity == nil && s.Affinity != nil {
		spec.Affinity = s.Affinity.DeepCopy()
	}
}

func hasToleration(tolerations []corev1.Toleration, target corev1.Toleration) bool {
	for _, toleration := range tolerations {
		if toleration.MatchToleration(&target) {
			return true
		}
	}
	return false
}
