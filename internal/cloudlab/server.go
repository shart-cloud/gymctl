package cloudlab

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// ─── Resource index ───────────────────────────────────────────────────────────

// Index maps kindKey → namespace → name → raw JSON.
// kindKey is the lowercase plural form (pods, services, …).
// Cluster-scoped resources (Namespace) use namespace = "".
type Index map[string]map[string]map[string]json.RawMessage

// BuildIndex indexes the raw resource maps from a Scenario.
func BuildIndex(resources []map[string]interface{}) (Index, error) {
	idx := make(Index)

	for _, res := range resources {
		apiVersion, _ := res["apiVersion"].(string)
		kind, _ := res["kind"].(string)
		if apiVersion == "" || kind == "" {
			continue
		}

		meta, _ := res["metadata"].(map[string]interface{})
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		if name == "" {
			continue
		}

		kindKey := pluralKind(kind)
		if kindKey == "namespaces" {
			ns = "" // cluster-scoped
		}

		raw, err := json.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("marshaling %s/%s: %w", kind, name, err)
		}

		if idx[kindKey] == nil {
			idx[kindKey] = make(map[string]map[string]json.RawMessage)
		}
		if idx[kindKey][ns] == nil {
			idx[kindKey][ns] = make(map[string]json.RawMessage)
		}
		idx[kindKey][ns][name] = json.RawMessage(raw)
	}

	return idx, nil
}

func pluralKind(kind string) string {
	switch strings.ToLower(kind) {
	case "pod":
		return "pods"
	case "service":
		return "services"
	case "configmap":
		return "configmaps"
	case "deployment":
		return "deployments"
	case "namespace":
		return "namespaces"
	default:
		return strings.ToLower(kind) + "s"
	}
}

func (idx Index) list(kindKey, ns string) []json.RawMessage {
	nsMap := idx[kindKey]
	if nsMap == nil {
		return nil
	}
	var items []json.RawMessage
	for namespace, nameMap := range nsMap {
		if ns != "" && namespace != ns {
			continue
		}
		for _, raw := range nameMap {
			items = append(items, raw)
		}
	}
	return items
}

func (idx Index) get(kindKey, ns, name string) (json.RawMessage, bool) {
	raw, ok := idx[kindKey][ns][name]
	return raw, ok
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

type listMeta struct {
	ResourceVersion string `json:"resourceVersion"`
}

type itemList struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   listMeta          `json:"metadata"`
	Items      []json.RawMessage `json:"items"`
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func writeList(w http.ResponseWriter, apiVersion, kind string, items []json.RawMessage) {
	if items == nil {
		items = []json.RawMessage{}
	}
	writeJSON(w, itemList{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata:   listMeta{ResourceVersion: "1"},
		Items:      items,
	})
}

func writeNotFound(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	writeJSON(w, map[string]interface{}{
		"kind":       "Status",
		"apiVersion": "v1",
		"status":     "Failure",
		"message":    message,
		"reason":     "NotFound",
		"code":       404,
	})
}

// ─── Discovery responses (static) ────────────────────────────────────────────

var (
	apiVersionsResponse = map[string]interface{}{
		"kind":       "APIVersions",
		"apiVersion": "v1",
		"versions":   []string{"v1"},
		"serverAddressByClientCIDRs": []map[string]string{
			{"clientCIDR": "0.0.0.0/0", "serverAddress": "localhost:6443"},
		},
	}

	coreResourceListResponse = map[string]interface{}{
		"kind":         "APIResourceList",
		"apiVersion":   "v1",
		"groupVersion": "v1",
		"resources": []map[string]interface{}{
			{"name": "namespaces", "namespaced": false, "kind": "Namespace", "verbs": []string{"get", "list"}},
			{"name": "pods", "namespaced": true, "kind": "Pod", "verbs": []string{"get", "list"}},
			{"name": "services", "namespaced": true, "kind": "Service", "verbs": []string{"get", "list"}},
			{"name": "configmaps", "namespaced": true, "kind": "ConfigMap", "verbs": []string{"get", "list"}},
		},
	}

	apiGroupListResponse = map[string]interface{}{
		"kind":       "APIGroupList",
		"apiVersion": "v1",
		"groups": []map[string]interface{}{
			{
				"name":             "apps",
				"versions":         []map[string]string{{"groupVersion": "apps/v1", "version": "v1"}},
				"preferredVersion": map[string]string{"groupVersion": "apps/v1", "version": "v1"},
			},
		},
	}

	appsGroupResponse = map[string]interface{}{
		"kind":       "APIGroup",
		"apiVersion": "v1",
		"name":       "apps",
		"versions":   []map[string]string{{"groupVersion": "apps/v1", "version": "v1"}},
		"preferredVersion": map[string]string{
			"groupVersion": "apps/v1",
			"version":      "v1",
		},
	}

	appsResourceListResponse = map[string]interface{}{
		"kind":         "APIResourceList",
		"apiVersion":   "v1",
		"groupVersion": "apps/v1",
		"resources": []map[string]interface{}{
			{"name": "deployments", "namespaced": true, "kind": "Deployment", "verbs": []string{"get", "list"}},
		},
	}
)

// ─── Server ───────────────────────────────────────────────────────────────────

type apiServer struct {
	idx      Index
	scenario *Scenario
}

// NewMux builds and returns an http.ServeMux wired to the given scenario index.
func NewMux(scenario *Scenario, idx Index) *http.ServeMux {
	s := &apiServer{idx: idx, scenario: scenario}
	mux := http.NewServeMux()

	// Discovery
	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiVersionsResponse)
	})
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, coreResourceListResponse)
	})
	mux.HandleFunc("GET /apis", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, apiGroupListResponse)
	})
	mux.HandleFunc("GET /apis/apps", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, appsGroupResponse)
	})
	mux.HandleFunc("GET /apis/apps/v1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, appsResourceListResponse)
	})

	// Namespaces (cluster-scoped)
	mux.HandleFunc("GET /api/v1/namespaces", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, "v1", "NamespaceList", s.idx.list("namespaces", ""))
	})
	mux.HandleFunc("GET /api/v1/namespaces/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		raw, ok := s.idx.get("namespaces", "", name)
		if !ok {
			writeNotFound(w, fmt.Sprintf("namespaces %q not found", name))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(raw) //nolint:errcheck
	})

	// Pods
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/pods", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, "v1", "PodList", s.idx.list("pods", r.PathValue("ns")))
	})
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/pods/{name}", func(w http.ResponseWriter, r *http.Request) {
		s.getResource(w, "pods", r.PathValue("ns"), r.PathValue("name"))
	})

	// Services
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/services", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, "v1", "ServiceList", s.idx.list("services", r.PathValue("ns")))
	})
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/services/{name}", func(w http.ResponseWriter, r *http.Request) {
		s.getResource(w, "services", r.PathValue("ns"), r.PathValue("name"))
	})

	// ConfigMaps
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/configmaps", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, "v1", "ConfigMapList", s.idx.list("configmaps", r.PathValue("ns")))
	})
	mux.HandleFunc("GET /api/v1/namespaces/{ns}/configmaps/{name}", func(w http.ResponseWriter, r *http.Request) {
		s.getResource(w, "configmaps", r.PathValue("ns"), r.PathValue("name"))
	})

	// Deployments (apps/v1)
	mux.HandleFunc("GET /apis/apps/v1/namespaces/{ns}/deployments", func(w http.ResponseWriter, r *http.Request) {
		writeList(w, "apps/v1", "DeploymentList", s.idx.list("deployments", r.PathValue("ns")))
	})
	mux.HandleFunc("GET /apis/apps/v1/namespaces/{ns}/deployments/{name}", func(w http.ResponseWriter, r *http.Request) {
		s.getResource(w, "deployments", r.PathValue("ns"), r.PathValue("name"))
	})

	return mux
}

func (s *apiServer) getResource(w http.ResponseWriter, kindKey, ns, name string) {
	raw, ok := s.idx.get(kindKey, ns, name)
	if !ok {
		writeNotFound(w, fmt.Sprintf("%s %q not found in namespace %q", kindKey, name, ns))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw) //nolint:errcheck
}

// Serve loads the scenario at scenarioPath and starts the mock API server on
// the given port. It blocks until the server exits.
func Serve(scenarioPath string, port int) error {
	scenario, err := Load(scenarioPath)
	if err != nil {
		return fmt.Errorf("loading scenario: %w", err)
	}

	idx, err := BuildIndex(scenario.Resources)
	if err != nil {
		return fmt.Errorf("building index: %w", err)
	}

	for kindKey, nsMap := range idx {
		total := 0
		for _, nameMap := range nsMap {
			total += len(nameMap)
		}
		log.Printf("indexed %d %s", total, kindKey)
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("gymctl serve: %s (%s) listening on %s", scenario.Meta.Title, scenario.Meta.ID, addr)
	return http.ListenAndServe(addr, NewMux(scenario, idx))
}
