package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"citytree/internal/application"
	"citytree/internal/persistence"
	"citytree/internal/selfcheck"
	"citytree/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Printf("启动失败: %v", err)
		os.Exit(1)
	}
}

func run() error {
	addrFlag := flag.String("addr", "", "回环监听地址，例如 127.0.0.1:19081")
	dbFlag := flag.String("db", "data/city-tree.db", "SQLite 数据库路径")
	selfcheckFlag := flag.Bool("selfcheck", false, "执行有界端到端自检后退出")
	flag.Parse()
	addr, err := web.ResolveAddr(*addrFlag)
	if err != nil {
		return err
	}
	dbPath := *dbFlag
	var cleanup func()
	if *selfcheckFlag {
		tempDir, err := os.MkdirTemp("", "citytree-selfcheck-*")
		if err != nil {
			return err
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }
		defer cleanup()
		dbPath = filepath.Join(tempDir, "selfcheck.db")
	}
	ctx := context.Background()
	store, err := persistence.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	service := application.NewService(store)
	handler := web.NewServer(service).Handler()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", addr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	if *selfcheckFlag {
		return runSelfcheck(server, listener, serveErr)
	}
	log.Printf("城市树木健康巡检台已启动: http://%s", listener.Addr().String())
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err = <-serveErr:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-signalCtx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func runSelfcheck(server *http.Server, listener net.Listener, serveErr <-chan error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	summary, err := selfcheck.Run(ctx, "http://"+listener.Addr().String())
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	select {
	case serveFailure := <-serveErr:
		if serveFailure != nil && serveFailure != http.ErrServerClosed && err == nil {
			err = serveFailure
		}
	default:
	}
	if err != nil {
		return err
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	data, _ := json.Marshal(summary)
	fmt.Printf("自检通过：批次闭环、幂等重试、版本冲突和凭据摘要均已验证\n%s\n", data)
	return nil
}
