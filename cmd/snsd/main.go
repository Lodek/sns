package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/lodek/sns/config"
	snsv1 "github.com/lodek/sns/gen/sns/v1"
	"github.com/lodek/sns/notify"
	"github.com/lodek/sns/server"
	"github.com/lodek/sns/store"
	"github.com/lodek/sns/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cfg := config.Load()

	st, err := store.New(cfg.BadgerDir)
	if err != nil {
		slog.Error("open store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	var notifiers []notify.Notifier
	if cfg.TelegramToken != "" && cfg.TelegramChatID != "" {
		notifiers = append(notifiers, notify.NewTelegram(cfg.TelegramToken, cfg.TelegramChatID))
		slog.Info("registered notifier", "backend", "telegram")
	}
	if cfg.DiscordWebhookURL != "" {
		notifiers = append(notifiers, notify.NewDiscord(cfg.DiscordWebhookURL))
		slog.Info("registered notifier", "backend", "discord")
	}
	if len(notifiers) == 0 {
		slog.Warn("no notification backends configured")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	w := worker.New(st, notifiers, cfg.WorkerInterval)
	go w.Run(ctx)

	srv := server.New(st)
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		slog.Error("listen", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	snsv1.RegisterAlertServiceServer(gs, srv)
	reflection.Register(gs)

	// Start gRPC gateway (HTTP proxy).
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := snsv1.RegisterAlertServiceHandlerFromEndpoint(ctx, mux, cfg.GRPCAddr, opts); err != nil {
		slog.Error("register gateway", "error", err)
		os.Exit(1)
	}
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
	go func() {
		slog.Info("HTTP gateway listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http serve", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down servers")
		httpSrv.Shutdown(context.Background())
		gs.GracefulStop()
	}()

	slog.Info("gRPC server listening", "addr", cfg.GRPCAddr)
	if err := gs.Serve(lis); err != nil {
		slog.Error("serve", "error", err)
		os.Exit(1)
	}
}
