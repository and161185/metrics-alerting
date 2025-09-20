package grpcsvc

import (
	"context"
	"errors"
	"fmt"

	metricsv1 "github.com/and161185/metrics-alerting/internal/api/metricsv1"
	"github.com/and161185/metrics-alerting/internal/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Storage interface {
	Ping(ctx context.Context) error

	UpdateGauge(ctx context.Context, id string, value float64) error
	UpdateCounter(ctx context.Context, id string, delta int64) error

	GetGauge(ctx context.Context, id string) (val float64, ok bool, err error)
	GetCounter(ctx context.Context, id string) (val int64, ok bool, err error)

	GetAll(ctx context.Context) (gauges map[string]float64, counters map[string]int64, err error)
}

type Service struct {
	metricsv1.UnimplementedMetricsServiceServer
	st Storage
}

func NewService(st Storage) *Service {
	return &Service{st: st}
}

func Register(server *grpc.Server, st Storage) {
	metricsv1.RegisterMetricsServiceServer(server, NewService(st))
}

func (s *Service) Ping(ctx context.Context, _ *metricsv1.Empty) (*metricsv1.Empty, error) {
	if err := s.st.Ping(ctx); err != nil {
		return nil, toStatus(err)
	}
	return &metricsv1.Empty{}, nil
}

func (s *Service) UpdateMetric(ctx context.Context, in *metricsv1.Metric) (*metricsv1.Empty, error) {
	if in == nil || in.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty metric.id")
	}
	switch in.Kind {
	case metricsv1.MetricKind_GAUGE:
		vg, ok := in.Value.(*metricsv1.Metric_Gauge)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "expected gauge value")
		}
		if err := s.st.UpdateGauge(ctx, in.Id, vg.Gauge); err != nil {
			return nil, toStatus(err)
		}
	case metricsv1.MetricKind_COUNTER:
		vc, ok := in.Value.(*metricsv1.Metric_Counter)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "expected counter value")
		}
		if err := s.st.UpdateCounter(ctx, in.Id, vc.Counter); err != nil {
			return nil, toStatus(err)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown metric kind")
	}
	return &metricsv1.Empty{}, nil
}

func (s *Service) UpdateBatch(ctx context.Context, in *metricsv1.MetricsBatch) (*metricsv1.Empty, error) {
	if in == nil || len(in.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty batch")
	}
	for i, m := range in.Items {
		if _, err := s.UpdateMetric(ctx, m); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "item %d: %v", i, err)
		}
	}
	return &metricsv1.Empty{}, nil
}

func (s *Service) GetMetric(ctx context.Context, in *metricsv1.GetRequest) (*metricsv1.Metric, error) {
	if in == nil || in.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id")
	}
	switch in.Kind {
	case metricsv1.MetricKind_GAUGE:
		val, ok, err := s.st.GetGauge(ctx, in.Id)
		if err != nil {
			return nil, toStatus(err)
		}
		if !ok {
			return nil, status.Error(codes.NotFound, "gauge not found")
		}
		return &metricsv1.Metric{
			Id:   in.Id,
			Kind: metricsv1.MetricKind_GAUGE,
			Value: &metricsv1.Metric_Gauge{
				Gauge: val,
			},
		}, nil

	case metricsv1.MetricKind_COUNTER:
		val, ok, err := s.st.GetCounter(ctx, in.Id)
		if err != nil {
			return nil, toStatus(err)
		}
		if !ok {
			return nil, status.Error(codes.NotFound, "counter not found")
		}
		return &metricsv1.Metric{
			Id:   in.Id,
			Kind: metricsv1.MetricKind_COUNTER,
			Value: &metricsv1.Metric_Counter{
				Counter: val,
			},
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown metric kind")
	}
}

func (s *Service) GetAll(ctx context.Context, _ *metricsv1.Empty) (*metricsv1.MetricsBatch, error) {
	gs, cs, err := s.st.GetAll(ctx)
	if err != nil {
		return nil, toStatus(err)
	}
	out := &metricsv1.MetricsBatch{Items: make([]*metricsv1.Metric, 0, len(gs)+len(cs))}
	for id, v := range gs {
		out.Items = append(out.Items, &metricsv1.Metric{
			Id:   id,
			Kind: metricsv1.MetricKind_GAUGE,
			Value: &metricsv1.Metric_Gauge{
				Gauge: v,
			},
		})
	}
	for id, v := range cs {
		out.Items = append(out.Items, &metricsv1.Metric{
			Id:   id,
			Kind: metricsv1.MetricKind_COUNTER,
			Value: &metricsv1.Metric_Counter{
				Counter: v,
			},
		})
	}
	return out, nil
}

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return status.Error(codes.DeadlineExceeded, err.Error())
	}
	switch {
	case errors.Is(err, errs.ErrMetricNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal: %v", err))
	}
}
