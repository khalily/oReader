package worker

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/service"
)

const (
	// DefaultShutdownTimeout is the maximum time to wait for in-progress refreshes to complete
	DefaultShutdownTimeout = 30 * time.Second
	// DefaultRefreshInterval is the default interval between refresh cycles
	DefaultRefreshInterval = 15 * time.Minute
)

// RefreshWorker manages background feed refresh operations
type RefreshWorker struct {
	cfg            *config.Config
	refreshService service.RefreshWorkerService
	ticker         *time.Ticker
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mu             sync.Mutex
	running        bool
}

// NewRefreshWorker creates a new background refresh worker
func NewRefreshWorker(cfg *config.Config, refreshService service.RefreshWorkerService) *RefreshWorker {
	return &RefreshWorker{
		cfg:            cfg,
		refreshService: refreshService,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the background refresh worker
// It performs an initial refresh and then starts the periodic ticker
func (w *RefreshWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		log.Warn().Msg("Refresh worker already running")
		return nil
	}
	w.running = true
	w.mu.Unlock()

	log.Info().Msg("Starting background refresh worker")

	// Perform initial refresh on startup
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.runInitialRefresh(ctx)
	}()

	// Get refresh interval from config
	interval, err := w.cfg.GetRefreshInterval()
	if err != nil {
		log.Warn().Err(err).Dur("interval", DefaultRefreshInterval).Msg("Invalid refresh interval, using default")
		interval = DefaultRefreshInterval
	}

	log.Info().Dur("interval", interval).Msg("Refresh interval configured")

	// Start periodic ticker
	w.ticker = time.NewTicker(interval)
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		w.runPeriodicRefresh(ctx)
	}()

	return nil
}

// Stop gracefully stops the background refresh worker
// It waits for in-progress refreshes to complete or times out
func (w *RefreshWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		log.Warn().Msg("Refresh worker not running")
		return
	}
	w.running = false
	w.mu.Unlock()

	log.Info().Msg("Stopping background refresh worker")

	// Stop the ticker
	if w.ticker != nil {
		w.ticker.Stop()
	}

	// Signal stop
	close(w.stopChan)

	// Wait for in-progress refreshes to complete or timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Background refresh worker stopped gracefully")
	case <-time.After(DefaultShutdownTimeout):
		log.Warn().Msg("Background refresh worker stop timed out")
	}
}

// WaitForShutdown blocks until SIGTERM or SIGINT is received
// and then gracefully shuts down the worker
func (w *RefreshWorker) WaitForShutdown() {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Stop the worker gracefully
	w.Stop()
}

// runInitialRefresh performs an initial refresh on startup
func (w *RefreshWorker) runInitialRefresh(ctx context.Context) {
	log.Info().Msg("Running initial feed refresh on startup")

	refreshCtx, cancel := context.WithTimeout(ctx, DefaultShutdownTimeout)
	defer cancel()

	result, err := w.refreshService.RefreshAllFeeds(refreshCtx)
	if err != nil {
		log.Error().Err(err).Msg("Initial refresh failed")
		return
	}

	log.Info().
		Int("total_feeds", result.TotalFeeds).
		Int("success_count", result.SuccessCount).
		Int("failure_count", result.FailureCount).
		Int("new_items", result.NewItems).
		Dur("duration", result.Duration).
		Msg("Initial refresh completed")
}

// runPeriodicRefresh runs the periodic refresh loop
func (w *RefreshWorker) runPeriodicRefresh(ctx context.Context) {
	for {
		select {
		case <-w.ticker.C:
			log.Info().Msg("Periodic refresh triggered")

			// Create a context with timeout for this refresh cycle
			refreshCtx, cancel := context.WithTimeout(ctx, DefaultShutdownTimeout)

			// Run refresh in a goroutine to avoid blocking the ticker
			go func(rctx context.Context) {
				defer cancel()
				result, err := w.refreshService.RefreshAllFeeds(rctx)
				if err != nil {
					log.Error().Err(err).Msg("Periodic refresh failed")
					return
				}

				log.Info().
					Int("total_feeds", result.TotalFeeds).
					Int("success_count", result.SuccessCount).
					Int("failure_count", result.FailureCount).
					Int("new_items", result.NewItems).
					Dur("duration", result.Duration).
					Msg("Periodic refresh completed")
			}(refreshCtx)

		case <-w.stopChan:
			log.Info().Msg("Periodic refresh loop stopped")
			return

		case <-ctx.Done():
			log.Info().Msg("Periodic refresh loop stopped due to context cancellation")
			return
		}
	}
}

// IsRunning returns true if the worker is currently running
func (w *RefreshWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// TriggerRefresh manually triggers a refresh cycle
// This can be used for admin endpoints or testing
func (w *RefreshWorker) TriggerRefresh(ctx context.Context) (*service.RefreshAllResult, error) {
	log.Info().Msg("Manual refresh triggered")
	return w.refreshService.RefreshAllFeeds(ctx)
}
