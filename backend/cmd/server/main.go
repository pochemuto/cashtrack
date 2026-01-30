package main

import (
	cashtrack "cashtrack/backend"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := cashtrack.InitializeApp(ctx)
	if err != nil {
		panic(err)
	}
	go app.Processor.Run(ctx, 10*time.Second)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		panic(err)
	}
}
