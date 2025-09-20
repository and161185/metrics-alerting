package client

import (
	"context"
	"net"
	"testing"
	"time"

	metricsv1 "github.com/and161185/metrics-alerting/internal/api/metricsv1"
	"github.com/and161185/metrics-alerting/internal/config"
	"github.com/and161185/metrics-alerting/model"
	"google.golang.org/grpc"
)

type testSvc struct {
	metricsv1.UnimplementedMetricsServiceServer
	got []*metricsv1.Metric
}

func (s *testSvc) Ping(ctx context.Context, _ *metricsv1.Empty) (*metricsv1.Empty, error) {
	return &metricsv1.Empty{}, nil
}
func (s *testSvc) UpdateMetric(ctx context.Context, m *metricsv1.Metric) (*metricsv1.Empty, error) {
	s.got = append(s.got, m)
	return &metricsv1.Empty{}, nil
}
func (s *testSvc) UpdateBatch(ctx context.Context, b *metricsv1.MetricsBatch) (*metricsv1.Empty, error) {
	s.got = append(s.got, b.Items...)
	return &metricsv1.Empty{}, nil
}
func (s *testSvc) GetMetric(ctx context.Context, r *metricsv1.GetRequest) (*metricsv1.Metric, error) {
	if r.Kind == metricsv1.MetricKind_GAUGE {
		return &metricsv1.Metric{Id: r.Id, Kind: r.Kind, Value: &metricsv1.Metric_Gauge{Gauge: 1}}, nil
	}
	return &metricsv1.Metric{Id: r.Id, Kind: r.Kind, Value: &metricsv1.Metric_Counter{Counter: 1}}, nil
}
func (s *testSvc) GetAll(context.Context, *metricsv1.Empty) (*metricsv1.MetricsBatch, error) {
	return &metricsv1.MetricsBatch{Items: s.got}, nil
}

func startTestGRPC(t *testing.T) (addr string, stop func(), svc *testSvc) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	s := grpc.NewServer()
	svc = &testSvc{}
	metricsv1.RegisterMetricsServiceServer(s, svc)

	go func() { _ = s.Serve(l) }()

	return l.Addr().String(), func() {
		s.GracefulStop()
		_ = l.Close()
	}, svc
}

type memStore struct {
	m map[string]*model.Metric
}

func newMem() *memStore { return &memStore{m: map[string]*model.Metric{}} }

func (m *memStore) Save(ctx context.Context, metric *model.Metric) error {
	m.m[metric.ID] = &model.Metric{ID: metric.ID, Type: metric.Type, Value: metric.Value, Delta: metric.Delta}
	return nil
}
func (m *memStore) GetAll(ctx context.Context) (map[string]*model.Metric, error) {
	return m.m, nil
}

func Test_Client_GRPC_Path(t *testing.T) {
	addr, stop, svc := startTestGRPC(t)
	defer stop()

	st := newMem()
	v := 3.14
	d := int64(7)
	_ = st.Save(context.Background(), &model.Metric{ID: "g", Type: model.Gauge, Value: &v})
	_ = st.Save(context.Background(), &model.Metric{ID: "c", Type: model.Counter, Delta: &d})

	cfg := &config.ClientConfig{
		GRPCEnable:      true,
		GRPCAddr:        addr,
		GRPCTLS:         false,
		GRPCDialTimeout: 3,
		GRPCCallTimeout: 3,
		ClientTimeout:   5,
	}

	cl, err := NewClient(st, cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() {
		if cl.grpcConn != nil {
			_ = cl.grpcConn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cl.sendMetricGRPC(ctx, &model.Metric{ID: "x", Type: model.Gauge, Value: &v}); err != nil {
		t.Fatalf("sendMetricGRPC: %v", err)
	}

	if err := cl.sendToServerGRPC(ctx); err != nil {
		t.Fatalf("sendToServerGRPC: %v", err)
	}

	if got := len(svc.got); got < 3 {
		t.Fatalf("server got %d items, want >=3", got)
	}
}

func Test_toProtoMetric_OK(t *testing.T) {
	v := 3.14
	m := &model.Metric{ID: "g", Type: model.Gauge, Value: &v}
	pm, err := toProtoMetric(m)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pm.GetGauge() != 3.14 {
		t.Fatalf("gauge=%v", pm.GetGauge())
	}

	d := int64(7)
	m2 := &model.Metric{ID: "c", Type: model.Counter, Delta: &d}
	pm2, err := toProtoMetric(m2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pm2.GetCounter() != 7 {
		t.Fatalf("counter=%v", pm2.GetCounter())
	}
}

func Test_toProtoMetric_Errors(t *testing.T) {
	if _, err := toProtoMetric(nil); err == nil {
		t.Fatal("want error on nil")
	}
	m := &model.Metric{ID: "", Type: model.Gauge}
	if _, err := toProtoMetric(m); err == nil {
		t.Fatal("want error on empty id")
	}
	m2 := &model.Metric{ID: "g", Type: model.Gauge} // no value
	if _, err := toProtoMetric(m2); err == nil {
		t.Fatal("want error on gauge without value")
	}
	m3 := &model.Metric{ID: "c", Type: model.Counter} // no delta
	if _, err := toProtoMetric(m3); err == nil {
		t.Fatal("want error on counter without delta")
	}
}
