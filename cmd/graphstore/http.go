package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"graphstore/internal/model"
	"graphstore/internal/rebuild"
	"graphstore/internal/store"
	"graphstore/internal/walk"
)

const webDir = "web"

type server struct {
	store   *store.Store
	walker  *walk.Walker
	rebuild *rebuild.Rebuilder
	dataDir string
	httpSrv *http.Server
}

func newServer(st *store.Store, dataDir string) *server {
	return &server{
		store:   st,
		walker:  walk.New(st, walk.Options{BatchSize: 3}),
		rebuild: rebuild.New(st, st.LabelIndex()),
		dataDir: dataDir,
	}
}

func (s *server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/vertices", s.handleVertices)
	mux.HandleFunc("/api/edges", s.handleEdges)
	mux.HandleFunc("/api/walk", s.handleWalk)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/labels", s.handleLabels)
	mux.HandleFunc("/api/rebuild", s.handleRebuild)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/", s.handleBrowse)

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return s.httpSrv.Serve(listener)
}

func (s *server) Shutdown(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	walInfo := map[string]any{"enabled": false}
	if walLog := s.store.WAL(); walLog != nil {
		walInfo = map[string]any{
			"enabled":    true,
			"segments":   walLog.SegmentCount(),
			"seq":        walLog.Seq(),
			"openHandles": walLog.OpenHandles(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"vertices":  s.store.VertexCount(),
		"edges":     s.store.EdgeCount(),
		"shards":    len(s.store.EdgeShards()),
		"buckets":   s.store.EdgeBuckets(),
		"indexedEdges": s.store.AttrIndex().EdgeCount(),
		"shardCapacity": s.store.ActiveShard(0).Capacity(),
		"wal":       walInfo,
		"timestamp": time.Now().Unix(),
	})
}

func (s *server) handleVertices(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		vertex, ok := s.store.GetVertex(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "vertex not found"})
			return
		}
		if err := s.store.DeleteVertex(id); err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": id, "label": vertex.Label})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"vertices": s.store.AllVertices(),
		"total":    s.store.VertexCount(),
	})
}

func (s *server) handleEdges(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		edge, ok := s.store.GetEdge(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "edge not found"})
			return
		}
		if err := s.store.DeleteEdge(id); err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted": id,
			"from":    edge.From,
			"to":      edge.To,
			"type":    edge.Type,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"edges": s.store.SortedEdgeIDs(),
		"total": s.store.EdgeCount(),
	})
}

func (s *server) handleWalk(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	query := r.URL.Query()
	start := query.Get("start")
	if start == "" {
		start = query.Get("label")
	}
	maxDepth := parsePositiveInt(query.Get("maxDepth"), 3)
	pageSize := parsePositiveInt(query.Get("pageSize"), 3)
	page := parseNonNegativeInt(query.Get("page"), 0)
	timeout := time.After(10 * time.Second)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	type result struct {
		paths  []*model.Path
		err    error
	}
	done := make(chan result, 1)
	go func() {
		var paths []*model.Path
		var err error
		if query.Get("label") != "" {
			paths, err = s.walker.WalkFromLabel(ctx, query.Get("label"), maxDepth)
		} else {
			paths, err = s.walker.Walk(ctx, start, maxDepth)
		}
		done <- result{paths: paths, err: err}
	}()

	select {
	case <-timeout:
		cancel()
		writeJSON(w, http.StatusGatewayTimeout, map[string]any{"error": "walk timed out"})
	case res := <-done:
		if res.err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": res.err.Error()})
			return
		}
		batches := s.walker.Batch(res.paths, pageSize)
		if len(batches) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"page":      0,
				"pages":     0,
				"total":     0,
				"batchSize": pageSize,
				"paths":     []*model.Path{},
			})
			return
		}
		if page >= len(batches) {
			page = 0
		}
		batch := batches[page]
		snapInfo := map[string]any{}
		if session := s.walker.Session(); session != nil && session.Snapshot() != nil {
			snap := session.Snapshot()
			snapInfo = map[string]any{
				"vertices":     snap.VertexCount,
				"shards":       snap.ShardCount,
				"edges":        snap.EdgeCount,
				"ageMillis":    snap.AgeMillis(time.Now().UnixNano()),
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"page":      page,
			"pages":     len(batches),
			"total":     len(res.paths),
			"batchSize": pageSize,
			"paths":     batch,
			"vertices":  walk.DistinctVertices(res.paths),
			"maxDepth":  walk.MaxDepthReached(res.paths),
			"snapshot":  snapInfo,
		})
	}
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	if key == "" || value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key and value are required"})
		return
	}
	edgeIDs := s.store.LookupByAttr(key, value)
	writeJSON(w, http.StatusOK, map[string]any{
		"key":    key,
		"value":  value,
		"edges":  edgeIDs,
		"total":  len(edgeIDs),
	})
}

func (s *server) handleLabels(w http.ResponseWriter, _ *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	writeJSON(w, http.StatusOK, map[string]any{
		"labels": s.store.LabelIndex().Labels(),
		"count":  s.store.LabelIndex().EntryCount(),
	})
}

func (s *server) handleRebuild(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	s.store.Metrics().IndexRebuilds.Inc()
	if err := s.rebuild.Rebuild(ctx); err != nil {
		s.store.Metrics().IndexRebuildErrors.Inc()
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"labels": s.store.LabelIndex().Labels(),
		"count":  s.store.LabelIndex().EntryCount(),
		"plan": map[string]any{
			"total":   s.rebuild.LastPlan().Total,
			"limit":   s.rebuild.LastPlan().IndexLimit,
			"labels":  s.rebuild.LastPlan().Labels,
			"feasible": s.rebuild.LastPlan().Feasible(),
		},
	})
}

func (s *server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	writeJSON(w, http.StatusOK, s.store.Metrics().Snapshot())
}

func (s *server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	s.store.Metrics().HTTPRequests.Inc()
	if r.URL.Path != "/" && r.URL.Path != "/web/browse.html" {
		http.NotFound(w, r)
		return
	}
	page := filepath.Join(webDir, "browse.html")
	if _, err := os.Stat(page); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "browse page missing"})
		return
	}
	http.ServeFile(w, r, page)
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseNonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
