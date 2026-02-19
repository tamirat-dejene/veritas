package main

import (
	"net/http"
	"os"

	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()
	zap.ReplaceGlobals(log)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	zap.L().Info("Service candidate-service starting", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}
