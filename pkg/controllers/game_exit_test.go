package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	gameContainer = "app"
	wolfContainer = "wolf"
)

func podWithStatuses(statuses ...corev1.ContainerStatus) corev1.Pod {
	return corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: statuses}}
}

func running(name string) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:  name,
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
	}
}

func terminated(name string) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:  name,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}},
	}
}

func restarted(name string) corev1.ContainerStatus {
	status := running(name)
	status.RestartCount = 1
	return status
}

func TestGameExited(t *testing.T) {
	gameContainers := sets.New(gameContainer)
	deleting := podWithStatuses(terminated(gameContainer))
	deleting.DeletionTimestamp = &metav1.Time{}

	tests := []struct {
		name string
		pods []corev1.Pod
		want bool
	}{
		{name: "no pods", pods: nil, want: false},
		{name: "game running", pods: []corev1.Pod{podWithStatuses(running(gameContainer), running(wolfContainer))}, want: false},
		{name: "game terminated", pods: []corev1.Pod{podWithStatuses(terminated(gameContainer), running(wolfContainer))}, want: true},
		{name: "game restarted", pods: []corev1.Pod{podWithStatuses(restarted(gameContainer), running(wolfContainer))}, want: true},
		{name: "only a sidecar exited", pods: []corev1.Pod{podWithStatuses(running(gameContainer), terminated(wolfContainer))}, want: false},
		{name: "pod already being deleted", pods: []corev1.Pod{deleting}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, gameExited(tt.pods, gameContainers))
		})
	}
}
