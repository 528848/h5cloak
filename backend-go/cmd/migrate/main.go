// 数据库迁移工具。
//
//	migrate up            执行全部未执行的迁移
//	migrate down [n]      回滚 n 步（默认 1）
//	migrate status        显示当前版本
//	migrate create <name> 生成一对空迁移文件
//
// 后续迭代会增加 import-v1 子命令用于 V1 数据迁移。
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"cloak/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	dir := migrationsDir()

	if cmd == "create" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		if err := create(dir, os.Args[2]); err != nil {
			fail(err)
		}
		return
	}

	cfg, err := config.Load("admin")
	if err != nil {
		fail(err)
	}
	m, err := migrate.New("file://"+filepath.ToSlash(dir), "mysql://"+stripParseTime(cfg.MySQL.DSN))
	if err != nil {
		fail(err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		n := 1
		if len(os.Args) > 2 {
			n, _ = strconv.Atoi(os.Args[2])
		}
		err = m.Steps(-n)
	case "status":
		v, dirty, verr := m.Version()
		if errors.Is(verr, migrate.ErrNilVersion) {
			fmt.Println("version: none")
			return
		}
		if verr != nil {
			fail(verr)
		}
		fmt.Printf("version: %d dirty: %v\n", v, dirty)
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fail(err)
	}
	fmt.Println("ok")
}

func migrationsDir() string {
	if d := os.Getenv("CLOAK_MIGRATIONS_DIR"); d != "" {
		return d
	}
	for _, c := range []string{"migrations", "../migrations", "../../migrations"} {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	abs, _ := filepath.Abs("migrations")
	return abs
}

func create(dir, name string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102150405")
	base := filepath.Join(dir, ts+"_"+strings.ReplaceAll(name, " ", "_"))
	for _, suffix := range []string{".up.sql", ".down.sql"} {
		if err := os.WriteFile(base+suffix, []byte("-- "+name+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Println("created", base+suffix)
	}
	return nil
}

// golang-migrate 的 mysql 驱动不认识 gorm DSN 里的 parseTime/loc，需要去掉。
func stripParseTime(dsn string) string {
	if i := strings.Index(dsn, "?"); i >= 0 {
		base, query := dsn[:i], dsn[i+1:]
		var kept []string
		for _, kv := range strings.Split(query, "&") {
			k := strings.SplitN(kv, "=", 2)[0]
			if k == "parseTime" || k == "loc" {
				continue
			}
			kept = append(kept, kv)
		}
		kept = append(kept, "multiStatements=true")
		return base + "?" + strings.Join(kept, "&")
	}
	return dsn + "?multiStatements=true"
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate up | down [n] | status | create <name>")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
