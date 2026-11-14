package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestWithUdevMount(t *testing.T) {
	udevMount := corev1.VolumeMount{Name: udevVolumeName, MountPath: udevMountPath}
	runtimeMount := corev1.VolumeMount{Name: "wolf-runtime", MountPath: "/tmp/.X11-unix"}
	hostUdevMount := corev1.VolumeMount{Name: "host-udev", MountPath: udevMountPath}

	tests := []struct {
		name   string
		mounts []corev1.VolumeMount
		want   []corev1.VolumeMount
	}{
		{name: "no mounts", mounts: nil, want: []corev1.VolumeMount{udevMount}},
		{
			name:   "appends to existing mounts",
			mounts: []corev1.VolumeMount{runtimeMount},
			want:   []corev1.VolumeMount{runtimeMount, udevMount},
		},
		{
			name:   "keeps a user provided /run/udev mount",
			mounts: []corev1.VolumeMount{hostUdevMount},
			want:   []corev1.VolumeMount{hostUdevMount},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, withUdevMount(tt.mounts))
		})
	}
}

func TestWithNetAdmin(t *testing.T) {
	t.Run("nil security context", func(t *testing.T) {
		got := withNetAdmin(nil)
		assert.Equal(t, []corev1.Capability{capabilityNetAdmin}, got.Capabilities.Add)
	})

	t.Run("keeps user settings without mutating them", func(t *testing.T) {
		userContext := &corev1.SecurityContext{
			RunAsUser:    ptr.To[int64](0),
			Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"SYS_NICE"}},
		}
		got := withNetAdmin(userContext)

		assert.Equal(t, []corev1.Capability{"SYS_NICE", capabilityNetAdmin}, got.Capabilities.Add)
		assert.Equal(t, ptr.To[int64](0), got.RunAsUser)
		assert.Equal(t, []corev1.Capability{"SYS_NICE"}, userContext.Capabilities.Add)
	})

	t.Run("does not duplicate NET_ADMIN", func(t *testing.T) {
		userContext := &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{Add: []corev1.Capability{capabilityNetAdmin}},
		}
		assert.Equal(t, []corev1.Capability{capabilityNetAdmin}, withNetAdmin(userContext).Capabilities.Add)
	})
}
