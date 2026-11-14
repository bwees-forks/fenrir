package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestWithDefaultEnv(t *testing.T) {
	defaults := []corev1.EnvVar{
		{Name: "LIBVA_DRIVER_NAME", Value: "nvidia"},
		{Name: "GST_DEBUG", Value: "2"},
	}

	t.Run("adds defaults that are not set", func(t *testing.T) {
		env := []corev1.EnvVar{{Name: "DISPLAY", Value: ":0"}}

		assert.Equal(t, []corev1.EnvVar{
			{Name: "DISPLAY", Value: ":0"},
			{Name: "LIBVA_DRIVER_NAME", Value: "nvidia"},
			{Name: "GST_DEBUG", Value: "2"},
		}, withDefaultEnv(env, defaults))
	})

	t.Run("app values win over defaults", func(t *testing.T) {
		env := []corev1.EnvVar{{Name: "LIBVA_DRIVER_NAME", Value: "radeonsi"}}

		assert.Equal(t, []corev1.EnvVar{
			{Name: "LIBVA_DRIVER_NAME", Value: "radeonsi"},
			{Name: "GST_DEBUG", Value: "2"},
		}, withDefaultEnv(env, defaults))
	})
}
