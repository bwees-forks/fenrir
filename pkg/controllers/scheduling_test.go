package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const gpuNodeLabel = "direwolf.games-on-whales.github.io/gpu"

var gpuToleration = corev1.Toleration{
	Key:      "nvidia.com/gpu",
	Operator: corev1.TolerationOpExists,
	Effect:   corev1.TaintEffectNoSchedule,
}

var gpuAffinity = &corev1.Affinity{
	NodeAffinity: &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{{
				MatchExpressions: []corev1.NodeSelectorRequirement{{
					Key:      gpuNodeLabel,
					Operator: corev1.NodeSelectorOpExists,
				}},
			}},
		},
	},
}

func TestParsePodScheduling(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    PodScheduling
		wantErr bool
	}{
		{name: "empty", raw: "", want: PodScheduling{}},
		{
			name: "json",
			raw:  `{"nodeSelector":{"` + gpuNodeLabel + `":"true"},"tolerations":[{"key":"nvidia.com/gpu","operator":"Exists","effect":"NoSchedule"}]}`,
			want: PodScheduling{
				NodeSelector: map[string]string{gpuNodeLabel: "true"},
				Tolerations:  []corev1.Toleration{gpuToleration},
			},
		},
		{
			name: "yaml",
			raw:  "nodeSelector:\n  " + gpuNodeLabel + ": \"true\"\n",
			want: PodScheduling{NodeSelector: map[string]string{gpuNodeLabel: "true"}},
		},
		{
			name: "helm chart defaults",
			raw:  `{"affinity":{},"nodeSelector":{},"tolerations":[]}`,
			want: PodScheduling{NodeSelector: map[string]string{}, Tolerations: []corev1.Toleration{}},
		},
		{name: "unknown field is rejected", raw: `{"nodeSelectors":{"a":"b"}}`, wantErr: true},
		{name: "malformed", raw: `{`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePodScheduling(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPodSchedulingApplyTo(t *testing.T) {
	scheduling := PodScheduling{
		NodeSelector: map[string]string{gpuNodeLabel: "true", "kubernetes.io/arch": "amd64"},
		Tolerations:  []corev1.Toleration{gpuToleration},
		Affinity:     gpuAffinity,
	}

	t.Run("fills an empty pod spec", func(t *testing.T) {
		spec := corev1.PodSpec{}
		scheduling.ApplyTo(&spec)

		assert.Equal(t, scheduling.NodeSelector, spec.NodeSelector)
		assert.Equal(t, []corev1.Toleration{gpuToleration}, spec.Tolerations)
		assert.Equal(t, gpuAffinity, spec.Affinity)
		assert.NotSame(t, gpuAffinity, spec.Affinity)
	})

	t.Run("app template values win", func(t *testing.T) {
		appAffinity := &corev1.Affinity{PodAntiAffinity: &corev1.PodAntiAffinity{}}
		spec := corev1.PodSpec{
			NodeSelector: map[string]string{"kubernetes.io/arch": "arm64"},
			Tolerations:  []corev1.Toleration{gpuToleration},
			Affinity:     appAffinity,
		}
		scheduling.ApplyTo(&spec)

		assert.Equal(t, map[string]string{gpuNodeLabel: "true", "kubernetes.io/arch": "arm64"}, spec.NodeSelector)
		assert.Equal(t, []corev1.Toleration{gpuToleration}, spec.Tolerations)
		assert.Same(t, appAffinity, spec.Affinity)
	})

	t.Run("does not mutate the operator defaults", func(t *testing.T) {
		spec := corev1.PodSpec{NodeSelector: map[string]string{"kubernetes.io/arch": "arm64"}}
		scheduling.ApplyTo(&spec)

		assert.Equal(t, "amd64", scheduling.NodeSelector["kubernetes.io/arch"])
	})

	t.Run("no constraints leaves spec untouched", func(t *testing.T) {
		spec := corev1.PodSpec{}
		PodScheduling{}.ApplyTo(&spec)

		assert.Equal(t, corev1.PodSpec{}, spec)
	})
}
