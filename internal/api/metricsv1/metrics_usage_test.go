package metricsv1

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	status "google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func copyGetRequest(dst *GetRequest, src *GetRequest) {
	dst.Id = src.Id
	dst.Kind = src.Kind
}

func copyMetric(dst *Metric, src *Metric) {
	dst.Id = src.Id
	dst.Kind = src.Kind
	switch v := src.Value.(type) {
	case *Metric_Gauge:
		dst.Value = &Metric_Gauge{Gauge: v.Gauge}
	case *Metric_Counter:
		dst.Value = &Metric_Counter{Counter: v.Counter}
	default:
		dst.Value = nil
	}
}

func copyBatch(dst *MetricsBatch, src *MetricsBatch) {
	dst.Items = make([]*Metric, len(src.Items))
	for i, m := range src.Items {
		n := &Metric{}
		copyMetric(n, m)
		dst.Items[i] = n
	}
}

type svc struct {
	UnimplementedMetricsServiceServer
}

func (svc) Ping(context.Context, *Empty) (*Empty, error)               { return &Empty{}, nil }
func (svc) UpdateMetric(context.Context, *Metric) (*Empty, error)      { return &Empty{}, nil }
func (svc) UpdateBatch(context.Context, *MetricsBatch) (*Empty, error) { return &Empty{}, nil }
func (svc) GetMetric(_ context.Context, r *GetRequest) (*Metric, error) {
	if r.Kind == MetricKind_GAUGE {
		return &Metric{Id: r.Id, Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 1.23}}, nil
	}
	return &Metric{Id: r.Id, Kind: MetricKind_COUNTER, Value: &Metric_Counter{Counter: 7}}, nil
}
func (svc) GetAll(context.Context, *Empty) (*MetricsBatch, error) {
	return &MetricsBatch{Items: []*Metric{
		{Id: "g", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 3.14}},
		{Id: "c", Kind: MetricKind_COUNTER, Value: &Metric_Counter{Counter: 42}},
	}}, nil
}

func bufServer(t *testing.T) (*grpc.Server, *grpc.ClientConn, func()) {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer()
	RegisterMetricsServiceServer(s, svc{})
	go s.Serve(lis)

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }

	cc, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	stop := func() { s.GracefulStop(); _ = lis.Close(); _ = cc.Close() }
	return s, cc, stop
}

func Test_ClientStubs_All(t *testing.T) {
	_, cc, stop := bufServer(t)
	defer stop()
	cl := NewMetricsServiceClient(cc)

	if _, err := cl.Ping(context.Background(), &Empty{}); err != nil {
		t.Fatal(err)
	}
	if _, err := cl.UpdateMetric(context.Background(),
		&Metric{Id: "x", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 5}},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := cl.UpdateBatch(context.Background(),
		&MetricsBatch{Items: []*Metric{
			{Id: "y", Kind: MetricKind_COUNTER, Value: &Metric_Counter{Counter: 1}},
		}},
	); err != nil {
		t.Fatal(err)
	}
	m, err := cl.GetMetric(context.Background(), &GetRequest{Id: "x", Kind: MetricKind_GAUGE})
	if err != nil {
		t.Fatal(err)
	}
	_ = m.GetId()
	_ = m.GetKind()
	_ = m.GetGauge()
	_ = m.GetCounter()
	all, err := cl.GetAll(context.Background(), &Empty{})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range all.GetItems() {
		_ = it.GetId()
		_ = it.GetKind()
		_, _ = it.GetValue().(*Metric_Gauge)
		_, _ = it.GetValue().(*Metric_Counter)
	}
}

type fakeSrvForHandlers struct {
	UnimplementedMetricsServiceServer
}

func (fakeSrvForHandlers) Ping(ctx context.Context, _ *Empty) (*Empty, error) { return &Empty{}, nil }
func (fakeSrvForHandlers) UpdateMetric(ctx context.Context, _ *Metric) (*Empty, error) {
	return &Empty{}, nil
}
func (fakeSrvForHandlers) UpdateBatch(ctx context.Context, _ *MetricsBatch) (*Empty, error) {
	return &Empty{}, nil
}
func (fakeSrvForHandlers) GetMetric(ctx context.Context, _ *GetRequest) (*Metric, error) {
	return &Metric{Id: "id", Kind: MetricKind_COUNTER, Value: &Metric_Counter{Counter: 1}}, nil
}
func (fakeSrvForHandlers) GetAll(ctx context.Context, _ *Empty) (*MetricsBatch, error) {
	return &MetricsBatch{Items: []*Metric{
		{Id: "g", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 3.14}},
	}}, nil
}

func Test_Handlers_WithInterceptor(t *testing.T) {
	srv := fakeSrvForHandlers{}
	called := 0
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
		called++
		return h(ctx, req)
	}
	{
		in := new(Empty)
		out, err := _MetricsService_Ping_Handler(srv, context.Background(), func(v interface{}) error { _ = in; return nil }, interceptor)
		if err != nil || out == nil || called == 0 {
			t.Fatalf("Ping interceptor failed: out=%v err=%v called=%d", out, err, called)
		}
	}
	{
		in := &Metric{Id: "x", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 5}}
		_, err := _MetricsService_UpdateMetric_Handler(srv, context.Background(), func(v interface{}) error { copyMetric(v.(*Metric), in); return nil }, interceptor)
		if err != nil {
			t.Fatalf("UpdateMetric interceptor err=%v", err)
		}
	}
	{
		in := &MetricsBatch{Items: []*Metric{{Id: "a", Kind: MetricKind_COUNTER, Value: &Metric_Counter{Counter: 2}}}}
		_, err := _MetricsService_UpdateBatch_Handler(srv, context.Background(), func(v interface{}) error { copyBatch(v.(*MetricsBatch), in); return nil }, interceptor)
		if err != nil {
			t.Fatalf("UpdateBatch interceptor err=%v", err)
		}
	}
	{
		in := &GetRequest{Id: "id", Kind: MetricKind_COUNTER}
		out, err := _MetricsService_GetMetric_Handler(srv, context.Background(), func(v interface{}) error { copyGetRequest(v.(*GetRequest), in); return nil }, interceptor)
		if err != nil {
			t.Fatalf("GetMetric interceptor err=%v", err)
		}
		if out.(*Metric).GetCounter() != 1 {
			t.Fatalf("GetMetric wrong counter: %v", out)
		}
	}
	{
		in := new(Empty)
		out, err := _MetricsService_GetAll_Handler(srv, context.Background(), func(v interface{}) error { _ = in; return nil }, interceptor)
		if err != nil || len(out.(*MetricsBatch).GetItems()) == 0 {
			t.Fatalf("GetAll interceptor err=%v out=%v", err, out)
		}
	}
}

func Test_Handlers_DecodeError(t *testing.T) {
	srv := fakeSrvForHandlers{}
	decErr := errors.New("dec boom")
	check := func(name string, fn func() error) {
		t.Helper()
		if err := fn(); !errors.Is(err, decErr) {
			t.Fatalf("%s: want %v, got %v", name, decErr, err)
		}
	}
	check("Ping", func() error {
		_, err := _MetricsService_Ping_Handler(srv, context.Background(), func(interface{}) error { return decErr }, nil)
		return err
	})
	check("UpdateMetric", func() error {
		_, err := _MetricsService_UpdateMetric_Handler(srv, context.Background(), func(interface{}) error { return decErr }, nil)
		return err
	})
	check("UpdateBatch", func() error {
		_, err := _MetricsService_UpdateBatch_Handler(srv, context.Background(), func(interface{}) error { return decErr }, nil)
		return err
	})
	check("GetMetric", func() error {
		_, err := _MetricsService_GetMetric_Handler(srv, context.Background(), func(interface{}) error { return decErr }, nil)
		return err
	})
	check("GetAll", func() error {
		_, err := _MetricsService_GetAll_Handler(srv, context.Background(), func(interface{}) error { return decErr }, nil)
		return err
	})
}

func Test_Unimplemented_Hooks_DirectCalls(t *testing.T) {
	var u UnimplementedMetricsServiceServer
	u.testEmbeddedByValue()
	u.mustEmbedUnimplementedMetricsServiceServer()
}

func Test_Proto_String_Getters(t *testing.T) {
	_ = MetricKind_COUNTER.String()

	var e Empty
	_ = e.String()

	m := &Metric{Id: "id", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 2.5}}
	_ = m.String()
	_ = m.GetId()
	_ = m.GetKind()
	_ = m.GetValue()
	_ = m.GetGauge()
	_ = m.GetCounter()

	b := &MetricsBatch{Items: []*Metric{m}}
	_ = b.String()
	_ = b.GetItems()

	gr := &GetRequest{Id: "x", Kind: MetricKind_COUNTER}
	_ = gr.String()
	_ = gr.GetId()
	_ = gr.GetKind()
}

func Test_Unimplemented_ReturnsUnimplemented(t *testing.T) {
	var u UnimplementedMetricsServiceServer
	ctx := context.Background()

	if _, err := u.Ping(ctx, &Empty{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("Ping: %v", err)
	}
	if _, err := u.UpdateMetric(ctx, &Metric{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("UpdateMetric: %v", err)
	}
	if _, err := u.UpdateBatch(ctx, &MetricsBatch{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("UpdateBatch: %v", err)
	}
	if _, err := u.GetMetric(ctx, &GetRequest{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("GetMetric: %v", err)
	}
	if _, err := u.GetAll(ctx, &Empty{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("GetAll: %v", err)
	}
}

func Test_Generated_Proto_Extras(t *testing.T) {
	_ = MetricKind_COUNTER.Enum()
	_ = MetricKind_COUNTER.Number()
	_ = MetricKind_COUNTER.Type()

	var e Empty
	e.ProtoMessage()

	m := &Metric{Id: "id", Kind: MetricKind_GAUGE, Value: &Metric_Gauge{Gauge: 2.5}}
	m.ProtoMessage()

	b := &MetricsBatch{Items: []*Metric{m}}
	b.ProtoMessage()

	gr := &GetRequest{Id: "x", Kind: MetricKind_COUNTER}
	gr.ProtoMessage()

	_ = file_v1_metrics_proto_rawDescGZIP()

	_ = m.GetId()
	_ = m.GetKind()
	_ = m.GetValue()
	_ = gr.GetId()
	_ = gr.GetKind()
}
