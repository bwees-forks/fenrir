package controllers

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// gpuDefaultEnv targets NVIDIA. Apps on other GPUs override these in their
// pod template, e.g. LIBVA_DRIVER_NAME=radeonsi for AMD.
var gpuDefaultEnv = []corev1.EnvVar{
	{Name: "LIBVA_DRIVER_NAME", Value: "nvidia"},
	{Name: "LD_LIBRARY_PATH", Value: "/usr/local/nvidia/lib:/usr/local/nvidia/lib64:/usr/local/lib"},
	{Name: "NVIDIA_DRIVER_CAPABILITIES", Value: "all"},
	{Name: "NVIDIA_VISIBLE_DEVICES", Value: "all"},
	{Name: "GST_VAAPI_ALL_DRIVERS", Value: "1"},
	{Name: "GST_DEBUG", Value: "2"},
}

func withDefaultEnv(env []corev1.EnvVar, defaults []corev1.EnvVar) []corev1.EnvVar {
	for _, fallback := range defaults {
		isSet := slices.ContainsFunc(env, func(e corev1.EnvVar) bool { return e.Name == fallback.Name })
		if !isSet {
			env = append(env, fallback)
		}
	}
	return env
}
