package main

import (
	"log/slog"
	"net/http"
	"os"

	"gitlab.com/twkod/my-space/apps/api/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	app := server.New(server.ConfigFromEnv(), logger)
	defer app.Close()

	logger.Info("starting api", "addr", app.Addr())
	if err := app.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}
