package controllers

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// wolf-agent publishes hotplugged input devices into /run/udev and announces
// them over netlink, which needs NET_ADMIN, so SDL/Steam in the app container
// can find them.
const (
	udevVolumeName = "direwolf-udev"
	udevMountPath  = "/run/udev"

	capabilityNetAdmin corev1.Capability = "NET_ADMIN"
)

func udevVolume() corev1.Volume {
	return corev1.Volume{
		Name:         udevVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
}

// withUdevMount leaves mounts alone when something is already mounted at
// /run/udev, so a user-provided hostPath keeps working.
func withUdevMount(mounts []corev1.VolumeMount) []corev1.VolumeMount {
	hasUdevMount := slices.ContainsFunc(mounts, func(m corev1.VolumeMount) bool {
		return m.MountPath == udevMountPath
	})
	if hasUdevMount {
		return mounts
	}
	return append(mounts, corev1.VolumeMount{Name: udevVolumeName, MountPath: udevMountPath})
}

func withNetAdmin(securityContext *corev1.SecurityContext) *corev1.SecurityContext {
	result := securityContext.DeepCopy()
	if result == nil {
		result = &corev1.SecurityContext{}
	}
	if result.Capabilities == nil {
		result.Capabilities = &corev1.Capabilities{}
	}
	if !slices.Contains(result.Capabilities.Add, capabilityNetAdmin) {
		result.Capabilities.Add = append(result.Capabilities.Add, capabilityNetAdmin)
	}
	return result
}
