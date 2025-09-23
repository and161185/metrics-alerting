package grpcsvc

import (
	"context"
	"errors"
	"net"
	"testing"

	metricsv1 "github.com/and161185/metrics-alerting/internal/api/metricsv1"
	"github.com/and161185/metrics-alerting/internal/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type storeOK struct{ fakeStore }

func Test_Server_Bufconn(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	Register(s, &storeOK{*newFake()})
	go s.Serve(lis)
	t.Cleanup(func() { s.GracefulStop(); lis.Close() })

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	cc, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer cc.Close()

	cl := metricsv1.NewMetricsServiceClient(cc)
	_, err = cl.UpdateBatch(context.Background(), &metricsv1.MetricsBatch{
		Items: []*metricsv1.Metric{
			{Id: "x", Kind: metricsv1.MetricKind_GAUGE, Value: &metricsv1.Metric_Gauge{Gauge: 1}},
		},
	})
	if err != nil {
		t.Fatalf("rpc: %v", err)
	}
}

func Test_toStatus_Mapping(t *testing.T) {
	if got := toStatus(nil); got != nil {
		t.Fatalf("nil -> %v", got)
	}

	tests := []struct {
		in   error
		want codes.Code
	}{
		{context.DeadlineExceeded, codes.DeadlineExceeded},
		{context.Canceled, codes.DeadlineExceeded},
		{errs.ErrMetricNotFound, codes.NotFound},
		{errors.New("boom"), codes.Internal},
	}

	for _, tc := range tests {
		got := toStatus(tc.in)
		if status.Code(got) != tc.want {
			t.Fatalf("%v -> %v, want %v", tc.in, status.Code(got), tc.want)
		}
	}
}
