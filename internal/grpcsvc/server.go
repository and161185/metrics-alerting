package grpcsvc

import (
	"context"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type Opts struct {
	Addr       string
	MaxRecvMsg int
	MaxSendMsg int
	TLS        bool
	CertFile   string
	KeyFile    string
	Logger     *zap.SugaredLogger
	Storage    Storage // из service.go
}

type Server struct {
	opts Opts
	gs   *grpc.Server
	ln   net.Listener
	errC chan error
}

func New(o Opts) (*Server, error) {
	ln, err := net.Listen("tcp", o.Addr)
	if err != nil {
		return nil, err
	}

	var sopts []grpc.ServerOption
	if o.MaxRecvMsg > 0 {
		sopts = append(sopts, grpc.MaxRecvMsgSize(o.MaxRecvMsg))
	}
	if o.MaxSendMsg > 0 {
		sopts = append(sopts, grpc.MaxSendMsgSize(o.MaxSendMsg))
	}
	if o.TLS {
		creds, err := credentials.NewServerTLSFromFile(o.CertFile, o.KeyFile)
		if err != nil {
			_ = ln.Close()
			return nil, err
		}
		sopts = append(sopts, grpc.Creds(creds))
	}

	sopts = append(sopts, grpc.ChainUnaryInterceptor(unaryLogging(o.Logger)))

	gs := grpc.NewServer(sopts...)
	Register(gs, o.Storage) // регаем сервис

	return &Server{
		opts: o,
		gs:   gs,
		ln:   ln,
		errC: make(chan error, 1),
	}, nil
}

func (s *Server) Start() {
	if s.opts.Logger != nil {
		s.opts.Logger.Infof("gRPC listening on %s", s.opts.Addr)
	}
	go func() {
		err := s.gs.Serve(s.ln)
		s.errC <- err
		close(s.errC)
	}()
}

func (s *Server) Err() <-chan error { return s.errC }

// Мягкая остановка с таймаутом.
func (s *Server) Stop(ctx context.Context) {
	done := make(chan struct{}, 1)
	go func() {
		s.gs.GracefulStop()
		_ = s.ln.Close()
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.gs.Stop() // жёстко
		_ = s.ln.Close()
	}
}

func unaryLogging(l *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	if l == nil {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		st := status.Convert(err)
		l.Infow("grpc",
			"method", info.FullMethod,
			"code", st.Code().String(),
			"dur", time.Since(start).String(),
		)
		return resp, err
	}
}
