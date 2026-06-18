package architect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	ciliumfake "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/fake"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEventStoreKeepsRecentEvents(t *testing.T) {
	store := NewEventStore(2)
	store.Publish(DashboardEvent{Type: "one"})
	store.Publish(DashboardEvent{Type: "two"})
	store.Publish(DashboardEvent{Type: "three"})

	events := store.Snapshot()
	require.Len(t, events, 2)
	assert.Equal(t, "two", events[0].Type)
	assert.Equal(t, "three", events[1].Type)
	assert.Less(t, events[0].ID, events[1].ID)
}

func TestDashboardState(t *testing.T) {
	mapping, err := service_mapping.NewFakeServiceMappingImpl()
	require.NoError(t, err)

	created := metav1.NewTime(time.Now().UTC().Add(-2 * time.Hour))
	ciliumClient := ciliumfake.NewSimpleClientset(&ciliumv2.CiliumNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "restrict-elasticsearch-access-seeder-123",
			Namespace:         "agora",
			CreationTimestamp: created,
			Annotations: map[string]string{
				IgnoreInGitOps:            "true",
				AnnotationUpdate:          created.Format(timeFormat),
				AnnotationSourceKind:      "Job",
				AnnotationSourceName:      "seeder-123",
				AnnotationSourceNamespace: "apologia",
			},
		},
	})

	handler := &PeriklesHandler{
		Mapping:      mapping,
		CrdName:      "test",
		CiliumClient: ciliumClient,
		Namespace:    "apologia",
		ElasticNs:    "agora",
		VaultNs:      "delphi",
		PendingUpdates: map[string][]MappingUpdate{
			"solon": {{HostName: "solon", ClientName: "api"}},
		},
		Events: NewEventStore(10),
	}
	handler.recordEvent("test.event", "Dashboard test", "", "", nil)

	request := httptest.NewRequest(http.MethodGet, "/perikles/v1/state", nil)
	response := httptest.NewRecorder()
	handler.DashboardHandler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	var state DashboardState
	require.NoError(t, json.NewDecoder(response.Body).Decode(&state))
	require.Len(t, state.Services, 1)
	require.Len(t, state.Policies, 1)
	require.Len(t, state.Events, 1)
	assert.Equal(t, "Job", state.Policies[0].SourceKind)
	assert.Equal(t, "seeder-123", state.Policies[0].SourceName)
	assert.Contains(t, state.PendingUpdates, "solon")
}

func TestDashboardPageAndHealth(t *testing.T) {
	handler := &PeriklesHandler{Events: NewEventStore(10)}

	t.Run("page", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		handler.DashboardHandler().ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Header().Get("Content-Type"), "text/html")
		assert.True(t, strings.Contains(response.Body.String(), "Περικλῆς"))
	})

	t.Run("health", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/perikles/v1/health", nil)
		response := httptest.NewRecorder()
		handler.DashboardHandler().ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	})

	t.Run("embedded assets", func(t *testing.T) {
		tests := []struct {
			path        string
			contentType string
			content     string
		}{
			{path: "/perikles/assets/dashboard.css", contentType: "text/css", content: ":root"},
			{path: "/perikles/assets/dashboard.js", contentType: "text/javascript", content: "EventSource"},
		}

		for _, test := range tests {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.DashboardHandler().ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			assert.Contains(t, response.Header().Get("Content-Type"), test.contentType)
			assert.Contains(t, response.Body.String(), test.content)
		}
	})

	t.Run("readonly", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/perikles/v1/state", nil)
		response := httptest.NewRecorder()
		handler.DashboardHandler().ServeHTTP(response, request)

		assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	})
}
