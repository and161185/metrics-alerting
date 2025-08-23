package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/and161185/metrics-alerting/internal/config"
	srv "github.com/and161185/metrics-alerting/internal/server"
	"github.com/and161185/metrics-alerting/internal/server/testutils"
	"github.com/and161185/metrics-alerting/internal/utils"
	"github.com/and161185/metrics-alerting/model"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateMetricHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{"invalid_method", http.MethodGet, "/update/gauge/test/1.23", http.StatusMethodNotAllowed},
		{"valid_gauge", http.MethodPost, "/update/gauge/test/1.23", http.StatusOK},
		{"valid_counter", http.MethodPost, "/update/counter/testCounter/1", http.StatusOK},
		{"invalid_counter_value", http.MethodPost, "/update/counter/testCounter/1.2", http.StatusBadRequest},
		{"invalid_type", http.MethodPost, "/update/type/testCounter/1", http.StatusBadRequest},
		{"invalid_url", http.MethodPost, "/update/gauge/gauge", http.StatusNotFound},
	}

	r := chi.NewRouter()
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	r.Post("/update/{type}/{name}/{value}", srv.UpdateMetricHandler)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestUpdateMetricHandlerJSON(t *testing.T) {
	tests := []struct {
		name       string
		metric     model.Metric
		wantStatus int
	}{
		{"valid_gauge", model.Metric{ID: "TestGauge", Type: "gauge", Value: utils.F64Ptr(42.0)}, http.StatusOK},
		{"valid_counter", model.Metric{ID: "TestCounter", Type: "counter", Delta: utils.I64Ptr(1)}, http.StatusOK},
		{"invalid_counter_value", model.Metric{ID: "TestCounter", Type: "counter"}, http.StatusBadRequest},
		{"invalid_type", model.Metric{ID: "TestGauge", Type: "invalid", Value: utils.F64Ptr(42.0)}, http.StatusBadRequest},
	}

	r := chi.NewRouter()

	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	r.Post("/update/", srv.UpdateMetricHandlerJSON)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.metric)
			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)})

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", srv.GetMetricHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "42", strings.TrimSpace(string(body)))
}

func TestGetMetricHandlerJSON(t *testing.T) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "test", Type: model.Gauge, Value: utils.F64Ptr(42.0)})

	r := chi.NewRouter()
	r.Post("/value/", srv.GetMetricHandlerJSON)

	body, _ := json.Marshal(model.Metric{ID: "test", Type: model.Gauge})
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var m model.Metric
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
	require.NotNil(t, m.Value)
	require.Equal(t, float64(42), *m.Value)
}

func TestListMetricsHandler(t *testing.T) {
	ctx := context.Background()
	srv := testutils.NewTestServer(ctx)
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "foo", Type: model.Gauge, Value: utils.F64Ptr(1.23)})
	_ = srv.Storage.Save(ctx, &model.Metric{ID: "bar", Type: model.Counter, Delta: utils.I64Ptr(10)})
	r := chi.NewRouter()
	r.Get("/", srv.ListMetricsHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(body), "foo")
	require.Contains(t, string(body), "bar")
}

func TestUpdateArrayMetricHandlerJSON(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		metrics     []model.Metric
		contentType string
		body        []byte
		wantStatus  int
	}{
		{
			"valid_metrics",
			[]model.Metric{{ID: "g", Type: "gauge", Value: utils.F64Ptr(42)}, {ID: "c", Type: "counter", Delta: utils.I64Ptr(1)}},
			"application/json", nil, http.StatusOK,
		},
		{"invalid_content_type", []model.Metric{{ID: "g", Type: "gauge", Value: utils.F64Ptr(42)}}, "text/plain", nil, http.StatusUnsupportedMediaType},
		{"invalid_json", nil, "application/json", []byte("{invalid}"), http.StatusBadRequest},
		{"empty_array", []model.Metric{}, "application/json", nil, http.StatusBadRequest},
		{"invalid_counter", []model.Metric{{ID: "c", Type: "counter"}}, "application/json", nil, http.StatusUnprocessableEntity},
	}

	r := chi.NewRouter()
	srv := testutils.NewTestServer(ctx)
	r.Post("/updates/", srv.UpdateArrayMetricHandlerJSON)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error
			if tc.body != nil {
				body = tc.body
			} else {
				body, err = json.Marshal(tc.metrics)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)
			if tc.wantStatus == http.StatusOK {
				for _, m := range tc.metrics {
					stored, err := srv.Storage.Get(ctx, &model.Metric{ID: m.ID, Type: m.Type})
					require.NoError(t, err)
					switch m.Type {
					case "gauge":
						require.Equal(t, *m.Value, *stored.Value)
					case "counter":
						require.Equal(t, *m.Delta, *stored.Delta)
					}
				}
			}
		})
	}
}

type memFS struct {
	saveCnt, loadCnt int
}

func (m *memFS) SaveToFile(_ context.Context, _ string) error   { m.saveCnt++; return nil }
func (m *memFS) LoadFromFile(_ context.Context, _ string) error { m.loadCnt++; return nil }

type stubStore struct {
	data map[string]*model.Metric
	err  error
}

func (s *stubStore) Save(ctx context.Context, m *model.Metric) error {
	if s.data == nil {
		s.data = map[string]*model.Metric{}
	}
	s.data[m.ID] = m
	return s.err
}
func (s *stubStore) SaveBatch(ctx context.Context, ms []model.Metric) error {
	for i := range ms {
		_ = s.Save(ctx, &ms[i])
	}
	return s.err
}
func (s *stubStore) Get(ctx context.Context, m *model.Metric) (*model.Metric, error) {
	v, ok := s.data[m.ID]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, s.err
}
func (s *stubStore) GetAll(ctx context.Context) (map[string]*model.Metric, error) {
	return s.data, s.err
}
func (s *stubStore) Ping(ctx context.Context) error { return s.err }

func TestNewServer_BuildRouter(t *testing.T) {
	cfg := &config.ServerConfig{Addr: "127.0.0.1:0"}
	s := srv.NewServer(&stubStore{}, cfg)
	require.NotNil(t, s)
	h := getRouterForTest(s)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestRun_StartStop(t *testing.T) {
	cfg := &config.ServerConfig{
		Addr:            freeAddr(t),
		StoreInterval:   1,
		FileStoragePath: "x.json",
		Restore:         true,
	}
	st := &stubStore{data: map[string]*model.Metric{}}
	s := srv.NewServer(st, cfg)

	fs := &memFS{}
	s.FileStore = fs

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()
	require.NoError(t, <-errCh)
	require.GreaterOrEqual(t, fs.loadCnt, 1)
}

func TestUpdateMetricHandlerJSON_Happy(t *testing.T) {
	cfg := &config.ServerConfig{Addr: "x"}
	s := srv.NewServer(&stubStore{data: map[string]*model.Metric{}}, cfg)

	r := chi.NewRouter()
	r.Post("/update", s.UpdateMetricHandlerJSON)

	body, _ := json.Marshal(model.Metric{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(1)})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	respBytes, _ := io.ReadAll(rr.Body)
	require.Contains(t, string(respBytes), `"id":"g"`)
}

func TestUpdateArrayMetricHandlerJSON_Happy(t *testing.T) {
	cfg := &config.ServerConfig{Addr: "x"}
	st := &stubStore{data: map[string]*model.Metric{}}
	s := srv.NewServer(st, cfg)

	r := chi.NewRouter()
	r.Post("/updates", s.UpdateArrayMetricHandlerJSON)

	arr := []model.Metric{
		{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(2)},
		{ID: "c", Type: model.Counter, Delta: utils.I64Ptr(3)},
	}
	body, _ := json.Marshal(arr)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 2.0, *st.data["g"].Value)
	require.EqualValues(t, 3, *st.data["c"].Delta)
}

func TestGetMetricHandler_Counter(t *testing.T) {
	cfg := &config.ServerConfig{Addr: "x"}
	st := &stubStore{data: map[string]*model.Metric{
		"c": {ID: "c", Type: model.Counter, Delta: utils.I64Ptr(9)},
	}}
	s := srv.NewServer(st, cfg)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", s.GetMetricHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/c", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "9", stringsTrim(rr.Body.String()))
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func getRouterForTest(s *srv.Server) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", s.UpdateMetricHandler)
	r.Post("/update", s.UpdateMetricHandlerJSON)
	r.Post("/updates", s.UpdateArrayMetricHandlerJSON)
	r.Get("/value/{type}/{name}", s.GetMetricHandler)
	r.Post("/value", s.GetMetricHandlerJSON)
	r.Get("/", s.ListMetricsHandler)
	r.Get("/ping", s.PingHandler)
	return r
}

func nopLogger() *zap.SugaredLogger { return zap.NewNop().Sugar() }

func newServerWithInMem(t *testing.T) *srv.Server {
	t.Helper()
	ctx := context.Background()
	s := testutils.NewTestServer(ctx)
	s.Config.Logger = nopLogger()
	s.Config.Key = ""
	return &s
}

func buildRouter(s *srv.Server) http.Handler {
	r := chi.NewRouter()
	r.Post("/update", s.UpdateMetricHandlerJSON)
	r.Post("/updates", s.UpdateArrayMetricHandlerJSON)
	r.Get("/value/{type}/{name}", s.GetMetricHandler)
	r.Post("/value", s.GetMetricHandlerJSON)
	r.Get("/", s.ListMetricsHandler)
	r.Get("/ping", s.PingHandler)
	return r
}

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func Test_NewServer_and_Router_MinSmoke(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func Test_Run_StartStop(t *testing.T) {
	s := newServerWithInMem(t)
	s.Config.Logger = nopLogger()
	s.Config.Restore = true
	s.Config.StoreInterval = 1
	s.Config.FileStoragePath = filepath.Join(t.TempDir(), "metrics.json")
	s.Config.Addr = freeAddr(t)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()

	if err := <-errCh; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func Test_UpdateMetricHandlerJSON_OK_and_Errors(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	body, _ := json.Marshal(model.Metric{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(1)})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader([]byte(`{}`)))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d", rr2.Code)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader([]byte(`{bad}`)))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr3.Code)
	}
}

func Test_UpdateArrayMetricHandlerJSON_OK_and_Errors(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	arr := []model.Metric{
		{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(2)},
		{ID: "c", Type: model.Counter, Delta: utils.I64Ptr(3)},
	}
	raw, _ := json.Marshal(arr)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}

	// Unsupported Media Type
	req2 := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte(`[]`)))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d", rr2.Code)
	}

	// Bad JSON
	req3 := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte(`{bad`)))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr3.Code)
	}

	// Empty array
	req4 := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader([]byte(`[]`)))
	req4.Header.Set("Content-Type", "application/json")
	rr4 := httptest.NewRecorder()
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr4.Code)
	}

	// Invalid counter (no delta) → 422
	bad := []byte(`[{"id":"c","type":"counter"}]`)
	req5 := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(bad))
	req5.Header.Set("Content-Type", "application/json")
	rr5 := httptest.NewRecorder()
	h.ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", rr5.Code)
	}
}

func Test_GetMetricHandler_Plain_Gauge_and_Counter(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	b1, _ := json.Marshal(model.Metric{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(4.2)})
	r1 := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(b1))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)

	b2, _ := json.Marshal(model.Metric{ID: "c", Type: model.Counter, Delta: utils.I64Ptr(7)})
	r2 := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(b2))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	// gauge
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/g", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "4.2" {
		t.Fatalf("body=%q", rr.Body.String())
	}

	// counter
	req2 := httptest.NewRequest(http.MethodGet, "/value/counter/c", nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("status=%d", rr2.Code)
	}
	if strings.TrimSpace(rr2.Body.String()) != "7" {
		t.Fatalf("body=%q", rr2.Body.String())
	}
}

func Test_GetMetricHandlerJSON_OK_and_NotFound(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	// gauge
	b, _ := json.Marshal(model.Metric{ID: "g", Type: model.Gauge, Value: utils.F64Ptr(10)})
	r := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	// ok
	body, _ := json.Marshal(model.Metric{ID: "g", Type: model.Gauge})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var got model.Metric
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.Value == nil || *got.Value != 10 {
		t.Fatalf("bad json: %+v", got)
	}

	// not found
	body2, _ := json.Marshal(model.Metric{ID: "absent", Type: model.Gauge})
	req2 := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr2.Code)
	}
}

func Test_ListMetricsHandler_HTML(t *testing.T) {
	s := newServerWithInMem(t)
	h := buildRouter(s)

	for _, m := range []model.Metric{
		{ID: "foo", Type: model.Gauge, Value: utils.F64Ptr(1)},
		{ID: "bar", Type: model.Counter, Delta: utils.I64Ptr(2)},
	} {
		raw, _ := json.Marshal(m)
		r := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	html, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(html), "foo") || !strings.Contains(string(html), "bar") {
		t.Fatalf("html: %s", string(html))
	}
}

type badPing struct{ srv.Storage }

func (b badPing) Ping(ctx context.Context) error {
	return context.DeadlineExceeded
}

func Test_PingHandler_OK_and_Error(t *testing.T) {
	// OK
	s := newServerWithInMem(t)
	h := buildRouter(s)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}

	// Error
	s2 := newServerWithInMem(t)
	s2.Storage = badPing{Storage: s.Storage}
	h2 := buildRouter(s2)
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr2 := httptest.NewRecorder()
	h2.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rr2.Code)
	}
}
