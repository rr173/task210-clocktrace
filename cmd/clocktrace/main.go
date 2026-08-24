// Command clocktrace 是网络时间同步根因定位服务的入口。
//
// 用法：
//
//	clocktrace --addr :8080 --db clocktrace.db   # 启动 HTTP 服务
//	clocktrace --smoke-test [--db smoke.db]      # 端到端自检（Docker CMD 判据）
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"task210-clocktrace/internal/httpapi"
	"task210-clocktrace/internal/service"
	"task210-clocktrace/internal/smoke"
	"task210-clocktrace/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "clocktrace.db", "SQLite database path")
	smokeMode := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smokeMode {
		smoke.Main([]string{*dbPath})
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app := service.New(db)
	srv := httpapi.New(app)

	log.Printf("network time-sync root-cause localization listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("serve: %v", err)
		os.Exit(1)
	}
}
