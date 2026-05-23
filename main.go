package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hvsio/ma-todo-res/config"
	"github.com/hvsio/ma-todo-res/manager"
	"github.com/hvsio/ma-todo-res/store"
	"github.com/hvsio/ma-todo-res/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	st := store.New(store.DefaultCapacity)
	mgr := manager.New(cfg, st)

	// Seed hashtags from env var — users can also add/remove via the UI.
	for _, tag := range cfg.Hashtags {
		if err := mgr.Add(tag); err != nil {
			log.Printf("warning: %v", err)
		}
	}

	tmplFS, err := fs.Sub(templateFS, "web/templates")
	if err != nil {
		log.Fatalf("template FS error: %v", err)
	}

	h, err := web.NewHandler(st, mgr, tmplFS, staticFS)
	if err != nil {
		log.Fatalf("handler error: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := web.New(":"+cfg.HTTPPort, mux)
	go func() {
		log.Printf("listening on :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down…")
	web.Shutdown(srv)
	mgr.Shutdown()
	log.Println("shutdown complete")
}
