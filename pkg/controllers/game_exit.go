package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	v1alpha1types "games-on-whales.github.io/direwolf/pkg/api/v1alpha1"
)

// gameExited reports whether any container from the App's own pod template
// has stopped or restarted in a live game pod. Wolf sidecars are ignored.
func gameExited(pods []corev1.Pod, gameContainers sets.Set[string]) bool {
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}
		for _, status := range pod.Status.ContainerStatuses {
			if !gameContainers.Has(status.Name) {
				continue
			}
			if status.RestartCount > 0 || status.State.Terminated != nil {
				return true
			}
		}
	}
	return false
}

func (c *SessionController) hasGameExited(ctx context.Context, session *v1alpha1types.Session) (bool, error) {
	app, err := c.AppInformer.Namespaced(session.Namespace).Get(session.Spec.GameReference.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get app: %w", err)
	}
	if app.Spec.Template == nil {
		return false, nil
	}

	gameContainers := sets.New[string]()
	for _, container := range app.Spec.Template.Spec.Containers {
		gameContainers.Insert(container.Name)
	}

	pods, err := c.K8sClient.CoreV1().Pods(session.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			AppLabel:  session.Spec.GameReference.Name,
			UserLabel: session.Spec.UserReference.Name,
		}).String(),
	})
	if err != nil {
		return false, fmt.Errorf("failed to list game pods: %w", err)
	}

	return gameExited(pods.Items, gameContainers), nil
}

// tearDownGame removes every session of the user's game along with its
// Deployment, so the pod is not restarted or recreated by a lingering session.
func (c *SessionController) tearDownGame(ctx context.Context, session *v1alpha1types.Session, ug userGame) error {
	sessionNames := sets.New(c.tracker.Sessions(ug)...).Insert(session.Name)

	for name := range sessionNames {
		if err := c.SessionClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete session %s/%s: %w", session.Namespace, name, err)
		}
	}

	if err := c.deleteDeployment(ctx, session.Namespace, ug); err != nil {
		return err
	}

	klog.Infof("Game %s for user %s exited, tore down %d session(s) and deployment %s",
		ug.Game, ug.User, sessionNames.Len(), ug.deploymentName())
	return nil
}
