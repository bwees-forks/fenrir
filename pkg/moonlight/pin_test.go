package moonlight

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1alpha1types "games-on-whales.github.io/direwolf/pkg/api/v1alpha1"
	"games-on-whales.github.io/direwolf/pkg/generic"
)

const testNamespace = "gaming"

func newPinTestServer(t *testing.T, usernames ...string) *RESTServer {
	t.Helper()
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	for _, name := range usernames {
		require.NoError(t, indexer.Add(&v1alpha1types.User{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		}))
	}
	return &RESTServer{
		manager:    &PairingManager{},
		UserLister: generic.NewLister[*v1alpha1types.User](indexer).Namespaced(testNamespace),
	}
}

func TestPinUsersHandler(t *testing.T) {
	server := newPinTestServer(t, "zoe", "bwees")
	recorder := httptest.NewRecorder()

	server.pinUsersHandler(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pin/users", http.NoBody))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `["bwees","zoe"]`, recorder.Body.String())
}

func TestPinHandlerPost(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantSent   *PinSubmission
	}{
		{
			name:       "known user",
			body:       `{"pin":"1234","secret":"secret","user":"bwees"}`,
			wantStatus: http.StatusOK,
			wantSent:   &PinSubmission{Pin: "1234", Username: "bwees"},
		},
		{
			name:       "unknown user",
			body:       `{"pin":"1234","secret":"secret","user":"alex"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing user",
			body:       `{"pin":"1234","secret":"secret"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newPinTestServer(t, "bwees")
			pins := make(chan PinSubmission, 1)
			server.manager.PendingPins.Store("secret", pendingPin{pins: pins})
			recorder := httptest.NewRecorder()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/pin/", strings.NewReader(tt.body))
			server.pinHandler(recorder, request)

			assert.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantSent == nil {
				assert.Empty(t, pins)
				return
			}
			assert.Equal(t, *tt.wantSent, <-pins)
		})
	}
}
