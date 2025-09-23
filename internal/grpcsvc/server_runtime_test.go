package grpcsvc

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubStore struct{}

func (stubStore) Ping(ctx context.Context) error                              { return nil }
func (stubStore) UpdateGauge(ctx context.Context, id string, v float64) error { return nil }
func (stubStore) UpdateCounter(ctx context.Context, id string, d int64) error { return nil }
func (stubStore) GetGauge(ctx context.Context, id string) (float64, bool, error) {
	return 0, false, nil
}
func (stubStore) GetCounter(ctx context.Context, id string) (int64, bool, error) {
	return 0, false, nil
}
func (stubStore) GetAll(ctx context.Context) (map[string]float64, map[string]int64, error) {
	return map[string]float64{}, map[string]int64{}, nil
}

func TestServer_StartStop_OK(t *testing.T) {
	s, err := New(Opts{
		Addr:    "127.0.0.1:0", // порт подберётся
		Logger:  zap.NewNop().Sugar(),
		Storage: stubStore{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s.Start()
	select {
	case <-s.Err(): // не должно сразу падать
		t.Fatal("serve exited too early")
	case <-time.After(20 * time.Millisecond):
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	s.Stop(ctx) // покрывает Stop/GracefulStop

	// канал должен закрыться/вернуть nil-ошибку
	select {
	case err := <-s.Err():
		if err != nil {
			// Serve обычно возвращает nil при GracefulStop
			t.Fatalf("Serve err: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Serve didn't exit")
	}
}

func TestUnaryLogging(t *testing.T) {
	logI := unaryLogging(zap.NewNop().Sugar())

	// хендлер, который вернёт ошибку InvalidArgument
	h := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "bad")
	}

	// сам интерсептор
	_, err := logI(context.Background(), struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/metrics.v1.MetricsService/UpdateMetric"},
		h,
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v", status.Code(err))
	}

	// ветка без логгера (nil)
	logI2 := unaryLogging(nil)
	if _, err := logI2(context.Background(), struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/x"}, h); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil logger path code = %v", status.Code(err))
	}
}
