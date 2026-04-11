package main

import (
	"log"
	"net/http"
	"path/filepath"

	"feishu-personality-agent/internal/config"
	"feishu-personality-agent/internal/server"
	"feishu-personality-agent/internal/session"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatal(err)
	}
	store := session.NewFileStore(filepath.Clean(cfg.AgentDataDir))
	srv := server.New(server.ServerConfig{App: cfg, Store: store})
	log.Printf("listening on http://%s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
