package shutdown

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/honeycarbs/project-ets/pkg/logging"
)

type Stoppable interface {
	Shutdown(ctx context.Context) error
}

func Graceful(signals []os.Signal, s Stoppable, timeout time.Duration, log *logging.Logger) {
	sigCtx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()

	<-sigCtx.Done()
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Warn("graceful shutdown completed with error", "err", err)
	} else {
		log.Info("graceful shutdown completed successfully")
	}
}
