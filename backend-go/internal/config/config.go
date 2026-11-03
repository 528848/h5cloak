// Package config 负责加载各服务的配置。
//
// 加载顺序：configs/<service>.yaml → 环境变量（前缀 CLOAK_，层级用下划线分隔，如 CLOAK_MYSQL_DSN）。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 四个服务共用的配置结构，各服务只读取自己关心的部分。
type Config struct {
	App       App       `mapstructure:"app"`
	HTTP      HTTP      `mapstructure:"http"`
	MySQL     MySQL     `mapstructure:"mysql"`
	Redis     Redis     `mapstructure:"redis"`
	Log       Log       `mapstructure:"log"`
	Auth      Auth      `mapstructure:"auth"`
	Gateway   Gateway   `mapstructure:"gateway"`
	Worker    Worker    `mapstructure:"worker"`
	Scheduler Scheduler `mapstructure:"scheduler"`
}

type App struct {
	Name      string `mapstructure:"name"`
	Env       string `mapstructure:"env"` // local, test, production
	NodeName  string `mapstructure:"node_name"`
	Timezone  string `mapstructure:"timezone"`
	PublicURL string `mapstructure:"public_url"` // 对外访问地址，用于拼接上传文件 URL
	UploadDir string `mapstructure:"upload_dir"` // 本地上传目录
}

type HTTP struct {
	Addr              string        `mapstructure:"addr"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	MetricsAddr       string        `mapstructure:"metrics_addr"`
}

type MySQL struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogSlow         time.Duration `mapstructure:"log_slow"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type Log struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

type Auth struct {
	JWTSecret    string        `mapstructure:"jwt_secret"`
	TokenTTL     time.Duration `mapstructure:"token_ttl"`
	LockAfter    int           `mapstructure:"lock_after"`
	LockDuration time.Duration `mapstructure:"lock_duration"`
}

type Gateway struct {
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

type Worker struct {
	Concurrency int `mapstructure:"concurrency"`
}

type Scheduler struct {
	LockKey string `mapstructure:"lock_key"`
}

// Load 读取 service 对应的配置文件。searchPaths 为空时默认搜索 ./configs 与 ../configs。
func Load(service string, searchPaths ...string) (*Config, error) {
	v := viper.New()
	v.SetConfigName(service)
	v.SetConfigType("yaml")
	if len(searchPaths) == 0 {
		searchPaths = []string{"./configs", "../configs", "../../configs"}
	}
	for _, p := range searchPaths {
		v.AddConfigPath(p)
	}
	v.SetEnvPrefix("CLOAK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v, service)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !asConfigNotFound(err, &notFound) {
			return nil, fmt.Errorf("read config %s: %w", service, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config %s: %w", service, err)
	}
	if cfg.App.Name == "" {
		cfg.App.Name = "cloak-" + service
	}
	return &cfg, nil
}

func asConfigNotFound(err error, target *viper.ConfigFileNotFoundError) bool {
	nf, ok := err.(viper.ConfigFileNotFoundError)
	if ok {
		*target = nf
	}
	return ok
}

func setDefaults(v *viper.Viper, service string) {
	v.SetDefault("app.env", "local")
	v.SetDefault("app.node_name", "local-1")
	v.SetDefault("app.timezone", "Asia/Shanghai")
	v.SetDefault("app.upload_dir", "./uploads")

	switch service {
	case "admin":
		v.SetDefault("http.addr", "127.0.0.1:8080")
		v.SetDefault("http.metrics_addr", "127.0.0.1:9080")
		v.SetDefault("app.public_url", "http://127.0.0.1:8080")
	case "gateway":
		v.SetDefault("http.addr", "127.0.0.1:8090")
		v.SetDefault("http.metrics_addr", "127.0.0.1:9090")
	case "worker":
		v.SetDefault("http.metrics_addr", "127.0.0.1:9091")
	case "scheduler":
		v.SetDefault("http.metrics_addr", "127.0.0.1:9092")
	}
	v.SetDefault("http.read_header_timeout", "10s")
	v.SetDefault("http.read_timeout", "30s")
	v.SetDefault("http.write_timeout", "60s")
	v.SetDefault("http.idle_timeout", "120s")
	v.SetDefault("http.shutdown_timeout", "15s")

	v.SetDefault("mysql.dsn", "cloak:cloak@tcp(127.0.0.1:3307)/cloak_v2?charset=utf8mb4&parseTime=true&loc=Local")
	v.SetDefault("mysql.max_open_conns", 50)
	v.SetDefault("mysql.max_idle_conns", 10)
	v.SetDefault("mysql.conn_max_lifetime", "1h")
	v.SetDefault("mysql.log_slow", "500ms")

	v.SetDefault("redis.addr", "127.0.0.1:6380")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 50)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	v.SetDefault("auth.token_ttl", "2h")
	v.SetDefault("auth.lock_after", 5)
	v.SetDefault("auth.lock_duration", "15m")

	v.SetDefault("worker.concurrency", 8)
	v.SetDefault("scheduler.lock_key", "scheduler:leader")
}
