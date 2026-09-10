package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/handler"
	"github.com/user/TinyProxyAPI/matcher"
	"github.com/user/TinyProxyAPI/signal"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 创建一次 handler，热重载时只更新其内部的 matcher 指针，
	// 不重建 handler（HTTP server 持有的 handler 实例保持不变）。
	m := matcher.NewMatcher(cfg)
	h := handler.NewProxyHandler(m)

	// 启动服务器
	go func() {
		log.Printf("server listening on %s", cfg.Server.Addr)
		if err := http.ListenAndServe(cfg.Server.Addr, h); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 合并两种重载触发：SIGHUP（Linux/Mac）+ 配置文件变化（跨平台）。
	// 任一来源触发都会通过同一个 channel 通知主循环重载。
	reload := make(chan struct{}, 1)

	hup := signal.Watch()
	go func() {
		for range hup {
			reload <- struct{}{}
		}
	}()

	if fileCh, err := signal.WatchConfig(*configPath); err != nil {
		log.Printf("warn: file watcher disabled: %v", err)
	} else {
		go func() {
			for range fileCh {
				reload <- struct{}{}
			}
		}()
	}

	for range reload {
		log.Println("reloading config...")
		newCfg, err := config.LoadConfig(*configPath)
		if err != nil {
			log.Printf("failed to reload config: %v", err)
			continue
		}
		// 复用现有 handler，仅原子替换其 matcher 指针。
		// 这样 HTTP server 持有的 handler 永远不变，但 ServeHTTP
		// 内部每次都会 Load 最新 matcher，新请求立即走新配置。
		h.Update(matcher.NewMatcher(newCfg))
		log.Println("config reloaded successfully")
	}
}