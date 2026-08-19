package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/port"
	"structured-log-alert-aggregator/internal/store"
	"structured-log-alert-aggregator/internal/transport"
	"structured-log-alert-aggregator/internal/worker"
)

func main() {
	serve := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := serve.String("addr", ":8080", "listen address")
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		slog.Info("usage: aggregator serve [-addr :8080]")
		return
	}
	_ = serve.Parse(os.Args[2:])
	var repo port.Repository = store.NewMemory()
	if url := os.Getenv("MYSQL_DSN"); url != "" {
		mysql, err := store.NewMySQL(context.Background(), url)
		if err != nil {
			slog.Error("database unavailable", "error", err)
			return
		}
		repo = mysql
	}
	service := app.NewWithRepository(repo)
	go (worker.Recovery{Repo: repo, Quiet: 5 * time.Minute}).Run(context.Background())
	go (worker.Notification{Repo: repo, Sender: worker.LogSender{}}).Run(context.Background())
	tokens := map[string]string{}
	for _, entry := range strings.Split(os.Getenv("API_TOKENS"), ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			tokens[parts[1]] = parts[0]
		}
	}
	if len(tokens) == 0 {
		tokens["demo-token"] = "demo"
	}
	server := transport.New(service, tokens)
	slog.Info("server started", "addr", *addr)
	if err := http.ListenAndServe(*addr, server); err != nil {
		slog.Error("server stopped", "error", err)
	}
}
