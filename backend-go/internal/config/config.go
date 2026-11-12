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
	Notify    Notify    `mapstructure:"notify"`
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
	SecretKey    string        `mapstructure:"secret_key"` // 站点密钥加密主密钥，admin 与 gateway 必须一致；空则用 jwt_secret
	TokenTTL     time.Duration `mapstructure:"token_ttl"`
	LockAfter    int           `mapstructure:"lock_after"`
	LockDuration time.Duration `mapstructure:"lock_duration"`
}

type Gateway struct {
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	EntryHost      string   `mapstructure:"entry_host"`
	Proxy          Proxy    `mapstructure:"proxy"`
	TLS            TLS      `mapstructure:"tls"`
	GeoIPPath      string   `mapstructure:"geoip_path"` // GeoLite2-Country.mmdb，空或不存在则不解析国家
	LogWALDir      string   `mapstructure:"log_wal_dir"`
}

// Proxy 反向代理参数。
type Proxy struct {
	MaxBodyBytes    int64         `mapstructure:"max_body_bytes"`    // 请求体上限，默认 32MB
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`      // 默认 5s
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"` // 默认 90s
	MaxIdlePerHost  int           `mapstructure:"max_idle_per_host"` // 默认 64
	ReviewNoIndex   bool          `mapstructure:"review_noindex"`    // 审核内容加 X-Robots-Tag
	InsecureOrigins bool          `mapstructure:"insecure_origins"`  // 跳过源站证书校验（本地自签源站）
	MaintenanceHTML string        `mapstructure:"maintenance_html"`  // 维护页文件路径，空用内置
	ReviewCacheTTL  time.Duration `mapstructure:"review_cache_ttl"`  // 审核页缓存时间，0 关闭，默认 5m
	ReviewCacheMB   int           `mapstructure:"review_cache_mb"`   // 审核页缓存总上限 MB，默认 128
}

// TLS 网关 HTTPS。mode: off | selfsigned | acme。
type TLS struct {
	Mode       string `mapstructure:"mode"`
	Addr       string `mapstructure:"addr"`        // HTTPS 监听，默认 :443；本地可 127.0.0.1:8443
	CertDir    string `mapstructure:"cert_dir"`    // 证书存储目录（自签与 ACME 共用）
	Email      string `mapstructure:"email"`       // ACME 账号邮箱
	Staging    bool   `mapstructure:"staging"`     // 使用 Let's Encrypt staging
	RedirectTo bool   `mapstructure:"redirect_to"` // HTTP 自动跳 HTTPS（ACME 挑战路径除外）
}

type Worker struct {
	Concurrency int    `mapstructure:"concurrency"`
	ExportDir   string `mapstructure:"export_dir"` // 导出文件目录（admin 与 worker 需共享）
}

// Notify 告警通知出口（worker 使用；admin 用于「发送测试」）。
type Notify struct {
	SMTP     SMTP     `mapstructure:"smtp"`
	Telegram Telegram `mapstructure:"telegram"`
	Webhook  Webhook  `mapstructure:"webhook"`
	// ConsoleURL 站内链接前缀（商户端地址），用于邮件/Telegram 中的跳转
	ConsoleURL string `mapstructure:"console_url"`
}

type SMTP struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	StartTLS bool   `mapstructure:"starttls"`
}

type Telegram struct {
	BotToken string `mapstructure:"bot_token"`
	APIBase  string `mapstructure:"api_base"` // 默认 https://api.telegram.org
}

// Webhook 投递策略。
type Webhook struct {
	Timeout     time.Duration `mapstructure:"timeout"`      // 默认 10s
	MaxAttempts int           `mapstructure:"max_attempts"` // 默认 5
	AllowHTTP   bool          `mapstructure:"allow_http"`   // 允许 http:// 目标（本地调试）
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
	if cfg.Auth.SecretKey == "" {
		cfg.Auth.SecretKey = cfg.Auth.JWTSecret
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
		v.SetDefault("gateway.proxy.max_body_bytes", 32<<20)
		v.SetDefault("gateway.proxy.dial_timeout", "5s")
		v.SetDefault("gateway.proxy.idle_conn_timeout", "90s")
		v.SetDefault("gateway.proxy.max_idle_per_host", 64)
		v.SetDefault("gateway.proxy.review_noindex", true)
		v.SetDefault("gateway.proxy.review_cache_ttl", "5m")
		v.SetDefault("gateway.proxy.review_cache_mb", 128)
		v.SetDefault("gateway.tls.mode", "off")
		v.SetDefault("gateway.tls.addr", ":443")
		v.SetDefault("gateway.tls.cert_dir", "./.local/certs")
		v.SetDefault("gateway.log_wal_dir", "./.local/wal")
		v.SetDefault("gateway.geoip_path", "./.local/GeoLite2-Country.mmdb")
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

	v.SetDefault("auth.jwt_secret", "local-dev-secret-change-me")
	v.SetDefault("auth.token_ttl", "2h")
	v.SetDefault("auth.lock_after", 5)
	v.SetDefault("auth.lock_duration", "15m")

	v.SetDefault("gateway.entry_host", "entry.cloak.local")
	v.SetDefault("worker.concurrency", 8)
	v.SetDefault("worker.export_dir", "./.local/exports")
	v.SetDefault("scheduler.lock_key", "scheduler:leader")
	v.SetDefault("notify.smtp.port", 587)
	v.SetDefault("notify.smtp.starttls", true)
	v.SetDefault("notify.telegram.api_base", "https://api.telegram.org")
	v.SetDefault("notify.webhook.timeout", "10s")
	v.SetDefault("notify.webhook.max_attempts", 5)
	v.SetDefault("notify.webhook.allow_http", false)
	v.SetDefault("notify.console_url", "http://localhost:5174")
}
