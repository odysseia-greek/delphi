package architect

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping/crd/v1alpha"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed dashboard_assets/*
var dashboardAssets embed.FS

type DashboardEvent struct {
	ID        uint64            `json:"id"`
	Time      time.Time         `json:"time"`
	Level     string            `json:"level"`
	Type      string            `json:"type"`
	Message   string            `json:"message"`
	Namespace string            `json:"namespace,omitempty"`
	Name      string            `json:"name,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

type EventStore struct {
	mu          sync.RWMutex
	capacity    int
	nextID      uint64
	events      []DashboardEvent
	subscribers map[chan DashboardEvent]struct{}
}

func NewEventStore(capacity int) *EventStore {
	if capacity < 1 {
		capacity = 1
	}
	return &EventStore{
		capacity:    capacity,
		events:      make([]DashboardEvent, 0, capacity),
		subscribers: make(map[chan DashboardEvent]struct{}),
	}
}

func (s *EventStore) Publish(event DashboardEvent) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.nextID++
	event.ID = s.nextID
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if event.Level == "" {
		event.Level = "info"
	}
	if len(s.events) == s.capacity {
		copy(s.events, s.events[1:])
		s.events[len(s.events)-1] = event
	} else {
		s.events = append(s.events, event)
	}
	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	s.mu.Unlock()
}

func (s *EventStore) Snapshot() []DashboardEvent {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]DashboardEvent(nil), s.events...)
}

func (s *EventStore) Subscribe() (<-chan DashboardEvent, func()) {
	channel := make(chan DashboardEvent, 16)
	if s == nil {
		close(channel)
		return channel, func() {}
	}

	s.mu.Lock()
	s.subscribers[channel] = struct{}{}
	s.mu.Unlock()

	return channel, func() {
		s.mu.Lock()
		if _, exists := s.subscribers[channel]; exists {
			delete(s.subscribers, channel)
			close(channel)
		}
		s.mu.Unlock()
	}
}

func (p *PeriklesHandler) recordEvent(eventType, message, namespace, name string, details map[string]string) {
	if p.Events == nil {
		return
	}
	p.Events.Publish(DashboardEvent{
		Type:      eventType,
		Message:   message,
		Namespace: namespace,
		Name:      name,
		Details:   details,
	})
}

type DashboardPolicy struct {
	Name            string    `json:"name"`
	Namespace       string    `json:"namespace"`
	Created         time.Time `json:"created"`
	AgeSeconds      int64     `json:"ageSeconds"`
	SourceKind      string    `json:"sourceKind,omitempty"`
	SourceName      string    `json:"sourceName,omitempty"`
	SourceNamespace string    `json:"sourceNamespace,omitempty"`
	Status          string    `json:"status,omitempty"`
}

type DashboardState struct {
	GeneratedAt    time.Time                  `json:"generatedAt"`
	Services       []v1alpha.Service          `json:"services"`
	PendingUpdates map[string][]MappingUpdate `json:"pendingUpdates"`
	Policies       []DashboardPolicy          `json:"policies"`
	Events         []DashboardEvent           `json:"events"`
	Errors         []string                   `json:"errors,omitempty"`
	Namespaces     []string                   `json:"namespaces"`
}

func (p *PeriklesHandler) DashboardHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.dashboardPage)
	mux.HandleFunc("/perikles/v1/dashboard", p.dashboardPage)
	mux.HandleFunc("/perikles/assets/dashboard.css", dashboardAsset("dashboard_assets/dashboard.css", "text/css; charset=utf-8"))
	mux.HandleFunc("/perikles/assets/dashboard.js", dashboardAsset("dashboard_assets/dashboard.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/perikles/v1/state", p.dashboardState)
	mux.HandleFunc("/perikles/v1/events", p.dashboardEvents)
	mux.HandleFunc("/perikles/v1/health", p.dashboardHealth)
	return mux
}

func (p *PeriklesHandler) StartDashboard() error {
	addr := strings.TrimSpace(p.DashboardAddr)
	if addr == "" {
		addr = ":8080"
	}
	p.recordEvent("dashboard.started", "Dashboard started", "", "", map[string]string{"address": addr})
	logging.System(fmt.Sprintf("Perikles dashboard listening on %s", addr))
	server := &http.Server{
		Addr:              addr,
		Handler:           p.DashboardHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

func (p *PeriklesHandler) dashboardPage(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" && request.URL.Path != "/perikles/v1/dashboard" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := dashboardAssets.ReadFile("dashboard_assets/dashboard.html")
	if err != nil {
		http.Error(writer, "dashboard unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = writer.Write(data)
}

func dashboardAsset(path, contentType string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.Header().Set("Allow", http.MethodGet)
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := dashboardAssets.ReadFile(path)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", contentType)
		writer.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = writer.Write(data)
	}
}

func (p *PeriklesHandler) dashboardState(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(p.buildDashboardState(request.Context()))
}

func (p *PeriklesHandler) dashboardHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}

func (p *PeriklesHandler) dashboardEvents(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	flusher.Flush()

	events, unsubscribe := p.Events.Subscribe()
	defer unsubscribe()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			writeServerEvent(writer, event)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(writer, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func writeServerEvent(writer http.ResponseWriter, event DashboardEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "id: %d\nevent: perikles\ndata: %s\n\n", event.ID, data)
}

func (p *PeriklesHandler) buildDashboardState(ctx context.Context) DashboardState {
	now := time.Now().UTC()
	state := DashboardState{
		GeneratedAt:    now,
		Services:       make([]v1alpha.Service, 0),
		PendingUpdates: make(map[string][]MappingUpdate),
		Policies:       make([]DashboardPolicy, 0),
		Events:         p.Events.Snapshot(),
		Errors:         make([]string, 0),
		Namespaces:     uniqueNamespaces(p.Namespace, p.ElasticNs, p.VaultNs),
	}
	if state.Events == nil {
		state.Events = make([]DashboardEvent, 0)
	}
	state.Namespaces = uniqueNamespaces(append(state.Namespaces, p.WatchedNamespaces...)...)

	if p.Mapping != nil {
		mapping, err := p.Mapping.Get(p.CrdName)
		if err != nil {
			state.Errors = append(state.Errors, fmt.Sprintf("service mapping: %v", err))
		} else {
			state.Services = append(state.Services, mapping.Spec.Services...)
			sort.Slice(state.Services, func(i, j int) bool {
				if state.Services[i].Namespace == state.Services[j].Namespace {
					return state.Services[i].Name < state.Services[j].Name
				}
				return state.Services[i].Namespace < state.Services[j].Namespace
			})
		}
	}

	p.Mutex.Lock()
	for host, updates := range p.PendingUpdates {
		state.PendingUpdates[host] = append([]MappingUpdate(nil), updates...)
	}
	p.Mutex.Unlock()

	if p.CiliumClient == nil {
		return state
	}
	for _, namespace := range state.Namespaces {
		policies, err := p.CiliumClient.CiliumV2().CiliumNetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			state.Errors = append(state.Errors, fmt.Sprintf("network policies in %s: %v", namespace, err))
			continue
		}
		for _, policy := range policies.Items {
			if !isPeriklesPolicy(policy.Name, policy.Annotations) {
				continue
			}
			created := policy.CreationTimestamp.Time
			dashboardPolicy := DashboardPolicy{
				Name:            policy.Name,
				Namespace:       namespace,
				Created:         created,
				SourceKind:      policy.Annotations[AnnotationSourceKind],
				SourceName:      policy.Annotations[AnnotationSourceName],
				SourceNamespace: policy.Annotations[AnnotationSourceNamespace],
			}
			if !created.IsZero() {
				dashboardPolicy.AgeSeconds = max(0, int64(now.Sub(created).Seconds()))
			}
			for _, condition := range policy.Status.Conditions {
				if condition.Type == "Valid" {
					dashboardPolicy.Status = string(condition.Status)
					break
				}
			}
			state.Policies = append(state.Policies, dashboardPolicy)
		}
	}
	sort.Slice(state.Policies, func(i, j int) bool {
		if state.Policies[i].Namespace == state.Policies[j].Namespace {
			return state.Policies[i].Name < state.Policies[j].Name
		}
		return state.Policies[i].Namespace < state.Policies[j].Namespace
	})
	return state
}

func isPeriklesPolicy(name string, annotations map[string]string) bool {
	if annotations[IgnoreInGitOps] == "true" && annotations[AnnotationUpdate] != "" {
		return true
	}
	return strings.HasPrefix(name, "restrict-elasticsearch-access-")
}
