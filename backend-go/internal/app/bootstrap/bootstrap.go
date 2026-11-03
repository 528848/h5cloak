// Package bootstrap 汇总每个服务启动时的公共步骤：配置、日志、指标注册表。
package bootstrap

import (
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap"

	"cloak/internal/config"
	"cloak/internal/pkg/logger"
)

// Runtime 服务公共运行时对象。
type Runtime struct {
	Service string
	Cfg     *config.Config
	Log     *zap.Logger
	Metrics *prometheus.Registry
}

// Init 加载配置与日志。失败时直接输出到 stderr 并退出，因为此时 logger 尚不可用。
func Init(service string) *Runtime {
	cfg, err := config.Load(service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log, err := logger.New(cfg.Log.Level, cfg.Log.Format, service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	log.Info("starting",
		zap.String("env", cfg.App.Env),
		zap.String("node", cfg.App.NodeName),
	)
	return &Runtime{Service: service, Cfg: cfg, Log: log, Metrics: reg}
}

// Close 刷新日志。
func (r *Runtime) Close() {
	_ = r.Log.Sync()
}
