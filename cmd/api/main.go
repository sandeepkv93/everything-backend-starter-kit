package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-oauth-rbac-service/internal/app"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := app.RunMigrationOnly(); err != nil {
			log.Fatal(err)
		}
		return
	}
	a, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		a.Logger.Info("server starting", "addr", a.Server.Addr)
		if err := a.Server.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = a.Server.Shutdown(ctx)
}
