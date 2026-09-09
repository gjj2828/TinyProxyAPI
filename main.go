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

	m := matcher.NewMatcher(cfg)
	h := handler.NewProxyHandler(m)

	// 启动服务器
	go func() {
		log.Printf("server listening on %s", cfg.Server.Addr)
		if err := http.ListenAndServe(cfg.Server.Addr, h); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 监听 SIGHUP
	hup := signal.Watch()
	for range hup {
		log.Println("received SIGHUP, reloading config...")
		newCfg, err := config.LoadConfig(*configPath)
		if err != nil {
			log.Printf("failed to reload config: %v", err)
			continue
		}
		m = matcher.NewMatcher(newCfg)
		h = handler.NewProxyHandler(m)
		log.Println("config reloaded successfully")
	}
}
