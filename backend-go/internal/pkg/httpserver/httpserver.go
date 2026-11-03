// Package httpserver 提供带优雅停机的 HTTP Server 与独立的指标端口。
package httpserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"cloak/internal/config"
)

// Server 包装业务 server 与指标 server。
type Server struct {
	log     *zap.Logger
	main    *http.Server
	metrics *http.Server
	cfg     config.HTTP
}

// New 创建 Server。handler 为空时不启动业务端口，只启动指标端口（worker/scheduler 用）。
func New(log *zap.Logger, cfg config.HTTP, handler http.Handler, reg *prometheus.Registry) *Server {
	s := &Server{log: log, cfg: cfg}
	if handler != nil && cfg.Addr != "" {
		s.main = &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		}
	}
	if cfg.MetricsAddr != "" && reg != nil {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		s.metrics = &http.Server{Addr: cfg.MetricsAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	}
	return s
}

// Run 启动并阻塞到收到退出信号，然后优雅停机。
func (s *Server) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	if s.main != nil {
		go func() {
			s.log.Info("http listening", zap.String("addr", s.main.Addr))
			if err := s.main.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}
	if s.metrics != nil {
		go func() {
			s.log.Info("metrics listening", zap.String("addr", s.metrics.Addr))
			if err := s.metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}

	select {
	case <-ctx.Done():
		s.log.Info("shutdown signal received")
	case err := <-errCh:
		s.log.Error("server error", zap.Error(err))
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	var firstErr error
	for _, srv := range []*http.Server{s.main, s.metrics} {
		if srv == nil {
			continue
		}
		if err := srv.Shutdown(shutdownCtx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
