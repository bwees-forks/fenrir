package controllers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	v1alpha1types "games-on-whales.github.io/direwolf/pkg/api/v1alpha1"
	generatedclient "games-on-whales.github.io/direwolf/pkg/generated/clientset/versioned/fake"
	generatedinformers "games-on-whales.github.io/direwolf/pkg/generated/informers/externalversions"
	"games-on-whales.github.io/direwolf/pkg/generic"
	"games-on-whales.github.io/direwolf/pkg/wolfapi"
)

const testNamespace = "games"

var testGame = userGame{User: "alice", Game: "steam"}

func testSession(name string, uid types.UID) *v1alpha1types.Session {
	return &v1alpha1types.Session{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace, UID: uid},
		Spec: v1alpha1types.SessionSpec{
			UserReference: v1alpha1types.UserReference{Name: testGame.User},
			GameReference: v1alpha1types.GameReference{Name: testGame.Game},
		},
	}
}

func testDeployment(owners ...metav1.OwnerReference) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testGame.deploymentName(),
			Namespace:       testNamespace,
			OwnerReferences: owners,
		},
	}
}

func newTestController(t *testing.T, k8sObjects []runtime.Object, sessions ...runtime.Object) (*SessionController, *k8sfake.Clientset) {
	t.Helper()

	fakeK8s := k8sfake.NewSimpleClientset(k8sObjects...)
	fakeDirewolf := generatedclient.NewSimpleClientset(sessions...)
	dwFactory := generatedinformers.NewSharedInformerFactory(fakeDirewolf, 0)
	k8sFactory := informers.NewSharedInformerFactory(fakeK8s, 0)

	controller := NewSessionController(
		fakeK8s,
		nil,
		nil,
		fakeDirewolf.DirewolfV1alpha1().Sessions(testNamespace),
		generic.NewInformer[*v1alpha1types.Session](dwFactory.Direwolf().V1alpha1().Sessions().Informer()),
		generic.NewInformer[*v1alpha1types.App](dwFactory.Direwolf().V1alpha1().Apps().Informer()),
		generic.NewInformer[*v1alpha1types.User](dwFactory.Direwolf().V1alpha1().Users().Informer()),
		generic.NewInformer[*appsv1.Deployment](k8sFactory.Apps().V1().Deployments().Informer()),
		SessionControllerOptions{},
	)

	stopCh := make(chan struct{})
	t.Cleanup(func() { close(stopCh) })
	dwFactory.Start(stopCh)
	k8sFactory.Start(stopCh)
	require.True(t, cacheWaitForSync(k8sFactory, dwFactory, 5*time.Second), "informers failed to sync")

	return controller, fakeK8s
}

func TestFindWolfSessionReturnsWolfSessionID(t *testing.T) {
	sessions := []wolfapi.Session{
		{AESKey: "other", AESIV: "other", ClientID: "1"},
		{AESKey: "key", AESIV: "iv", ClientID: "42"},
	}

	got, found := findWolfSession(sessions, &v1alpha1types.SessionInfo{AESKey: "key", AESIV: "iv"})
	assert.True(t, found)
	assert.Equal(t, "42", got.ClientID)

	_, found = findWolfSession(sessions, &v1alpha1types.SessionInfo{AESKey: "new", AESIV: "iv"})
	assert.False(t, found)
}

func TestReleaseGameDeletesDeploymentWithLastSession(t *testing.T) {
	controller, fakeK8s := newTestController(t, []runtime.Object{testDeployment()})
	controller.tracker.Track("first", testGame)
	controller.tracker.Track("second", testGame)
	deployments := fakeK8s.AppsV1().Deployments(testNamespace)

	require.NoError(t, controller.releaseGame(context.Background(), testNamespace, "first"))
	_, err := deployments.Get(context.Background(), testGame.deploymentName(), metav1.GetOptions{})
	require.NoError(t, err, "deployment deleted while a session still uses it")

	require.NoError(t, controller.releaseGame(context.Background(), testNamespace, "second"))
	_, err = deployments.Get(context.Background(), testGame.deploymentName(), metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err), "deployment still exists after its last session, err=%v", err)
}

func TestAdoptDeploymentAddsSessionsAsNonControllerOwners(t *testing.T) {
	creator := testSession("creator", "uid-creator")
	joiner := testSession("joiner", "uid-joiner")
	deployment := testDeployment(sessionOwnerReference(creator, true))

	controller, fakeK8s := newTestController(t, []runtime.Object{deployment}, creator, joiner)
	controller.tracker.Track(creator.Name, testGame)
	controller.tracker.Track(joiner.Name, testGame)

	require.NoError(t, controller.adoptDeployment(context.Background(), deployment, testGame))

	applied := appliedDeployment(t, fakeK8s)
	require.Len(t, applied.OwnerReferences, 1)
	owner := applied.OwnerReferences[0]
	assert.Equal(t, joiner.Name, owner.Name)
	assert.Equal(t, joiner.UID, owner.UID)
	assert.False(t, ptr.Deref(owner.Controller, false), "only the creating session may be the controller")
}

func TestAdoptDeploymentSkipsApplyWhenAllSessionsOwn(t *testing.T) {
	creator := testSession("creator", "uid-creator")
	deployment := testDeployment(sessionOwnerReference(creator, true))

	controller, fakeK8s := newTestController(t, []runtime.Object{deployment}, creator)
	controller.tracker.Track(creator.Name, testGame)
	fakeK8s.ClearActions()

	require.NoError(t, controller.adoptDeployment(context.Background(), deployment, testGame))
	assert.Empty(t, fakeK8s.Actions())
}

func TestFinalizeSessionReleasesSessionWithoutWolfStream(t *testing.T) {
	session := testSession("quitting", "uid-quitting")
	session.Finalizers = []string{v1alpha1types.SessionStreamFinalizer}
	session.DeletionTimestamp = ptr.To(metav1.Now())

	controller, _ := newTestController(t, nil, session)
	sessions := controller.SessionClient

	require.NoError(t, controller.finalizeSession(context.Background(), session))

	got, err := sessions.Get(context.Background(), session.Name, metav1.GetOptions{})
	if err == nil {
		assert.NotContains(t, got.Finalizers, v1alpha1types.SessionStreamFinalizer)
	} else {
		assert.True(t, errors.IsNotFound(err), "unexpected error: %v", err)
	}
}

func appliedDeployment(t *testing.T, fakeK8s *k8sfake.Clientset) appsv1.Deployment {
	t.Helper()
	for _, action := range fakeK8s.Actions() {
		patch, ok := action.(k8stesting.PatchAction)
		if !ok || patch.GetPatchType() != types.ApplyPatchType {
			continue
		}
		var deployment appsv1.Deployment
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &deployment))
		return deployment
	}
	t.Fatal("no apply patch was sent")
	return appsv1.Deployment{}
}
