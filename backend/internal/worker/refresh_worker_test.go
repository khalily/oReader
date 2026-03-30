package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/khalily/oreader/internal/config"
	"github.com/khalily/oreader/internal/model"
	"github.com/khalily/oreader/internal/service"
)

// mockRefreshService is a mock implementation of RefreshWorkerService
type mockRefreshService struct {
	mu           sync.Mutex
	refreshCount int
	lastResult   *service.RefreshAllResult
	refreshError error
	delay        time.Duration
}

func (m *mockRefreshService) RefreshAllFeeds(ctx context.Context) (*service.RefreshAllResult, error) {
	m.mu.Lock()
	m.refreshCount++
	m.mu.Unlock()
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.refreshError != nil {
		return nil, m.refreshError
	}
	if m.lastResult != nil {
		return m.lastResult, nil
	}
	return &service.RefreshAllResult{
		TotalFeeds:   1,
		SuccessCount: 1,
		Duration:     100 * time.Millisecond,
	}, nil
}

func (m *mockRefreshService) RefreshSingleFeed(ctx context.Context, feed *model.Feed) (service.SingleFeedResult, error) {
	return service.SingleFeedResult{}, nil
}

// TestRefreshWorker_StartStop tests basic start and stop behavior
func TestRefreshWorker_StartStop(t *testing.T) {
	ctx := context.Background()
	mockService := &mockRefreshService{}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "1s", // Short interval for testing
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	if !worker.IsRunning() {
		t.Error("Expected worker to be running")
	}

	// Wait a bit for initial refresh
	time.Sleep(200 * time.Millisecond)

	// Stop the worker
	worker.Stop()

	if worker.IsRunning() {
		t.Error("Expected worker to be stopped")
	}

	// Verify at least the initial refresh was called
	mockService.mu.Lock()
	count := mockService.refreshCount
	mockService.mu.Unlock()
	if count == 0 {
		t.Error("Expected at least one refresh call")
	}
}

// TestRefreshWorker_StartTwice tests that starting twice doesn't cause issues
func TestRefreshWorker_StartTwice(t *testing.T) {
	ctx := context.Background()
	mockService := &mockRefreshService{}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "1s",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Start again - should be idempotent
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Second start should not error: %v", err)
	}

	worker.Stop()
}

// TestRefreshWorker_InitialRefresh tests that initial refresh happens on startup
func TestRefreshWorker_InitialRefresh(t *testing.T) {
	ctx := context.Background()

	mockService := &mockRefreshService{
		lastResult: &service.RefreshAllResult{
			TotalFeeds:   5,
			SuccessCount: 5,
			NewItems:     10,
			Duration:     500 * time.Millisecond,
		},
	}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "1h", // Long interval to avoid periodic refresh during test
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Wait for initial refresh
	time.Sleep(700 * time.Millisecond)

	// Verify initial refresh was called
	mockService.mu.Lock()
	count := mockService.refreshCount
	mockService.mu.Unlock()
	if count != 1 {
		t.Errorf("Expected 1 refresh (initial), got %d", count)
	}
}

// TestRefreshWorker_PeriodicRefresh tests periodic refresh behavior
func TestRefreshWorker_PeriodicRefresh(t *testing.T) {
	ctx := context.Background()
	mockService := &mockRefreshService{}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "300ms", // Very short interval for testing
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Wait for multiple refreshes (initial + at least 2 periodic)
	time.Sleep(1 * time.Second)

	// Should have at least 3 refreshes (initial + 2+ periodic)
	mockService.mu.Lock()
	count := mockService.refreshCount
	mockService.mu.Unlock()
	if count < 3 {
		t.Errorf("Expected at least 3 refreshes, got %d", count)
	}
}

// TestRefreshWorker_GracefulShutdown tests graceful shutdown behavior
func TestRefreshWorker_GracefulShutdown(t *testing.T) {
	ctx := context.Background()

	mockService := &mockRefreshService{
		delay: 100 * time.Millisecond, // Slow refresh
	}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "10s",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Wait a bit then stop
	time.Sleep(50 * time.Millisecond)

	startTime := time.Now()
	worker.Stop()
	duration := time.Since(startTime)

	// Should complete within reasonable time (less than shutdown timeout)
	if duration > 2*time.Second {
		t.Errorf("Stop took too long: %v", duration)
	}

	// Stop should be idempotent
	worker.Stop()
}

// TestRefreshWorker_TriggerRefresh tests manual refresh triggering
func TestRefreshWorker_TriggerRefresh(t *testing.T) {
	ctx := context.Background()

	mockService := &mockRefreshService{
		lastResult: &service.RefreshAllResult{
			TotalFeeds:   3,
			SuccessCount: 3,
			NewItems:     7,
			Duration:     200 * time.Millisecond,
		},
	}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "1h",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Don't start the worker, just test manual trigger
	result, err := worker.TriggerRefresh(ctx)
	if err != nil {
		t.Fatalf("TriggerRefresh failed: %v", err)
	}

	if result.TotalFeeds != 3 {
		t.Errorf("Expected 3 total feeds, got %d", result.TotalFeeds)
	}

	mockService.mu.Lock()
	count := mockService.refreshCount
	mockService.mu.Unlock()
	if count != 1 {
		t.Errorf("Expected 1 refresh call, got %d", count)
	}
}

// TestRefreshWorker_RefreshError tests error handling during refresh
func TestRefreshWorker_RefreshError(t *testing.T) {
	ctx := context.Background()

	mockService := &mockRefreshService{
		refreshError: errors.New("refresh failed"),
	}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "1h",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start should succeed even if refresh fails
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Wait for initial refresh attempt
	time.Sleep(200 * time.Millisecond)

	// Worker should still be running despite error
	if !worker.IsRunning() {
		t.Error("Expected worker to still be running after refresh error")
	}
}

// TestRefreshWorker_ContextCancellation tests context cancellation handling
func TestRefreshWorker_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockService := &mockRefreshService{
		delay: 500 * time.Millisecond,
	}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "100ms",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Start the worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Wait a bit then cancel context
	time.Sleep(150 * time.Millisecond)
	cancel()

	// Give worker time to handle cancellation
	time.Sleep(100 * time.Millisecond)

	// Worker should handle context cancellation gracefully
	// The periodic loop should exit
}

// TestRefreshWorker_InvalidInterval tests handling of invalid refresh interval
func TestRefreshWorker_InvalidInterval(t *testing.T) {
	ctx := context.Background()

	mockService := &mockRefreshService{}

	cfg := &config.Config{
		Refresh: config.RefreshConfig{
			Interval: "invalid",
		},
	}

	worker := NewRefreshWorker(cfg, mockService)

	// Should fall back to default interval
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start should succeed with fallback interval: %v", err)
	}
	defer worker.Stop()

	// Should still run
	time.Sleep(100 * time.Millisecond)

	if !worker.IsRunning() {
		t.Error("Expected worker to be running with default interval")
	}
}
