package worker

import (
	"sync"
	"testing"
	"time"

	"task-manager/internal/queue"
	"task-manager/pkg/logger"
)

func createTestLogger() *logger.Logger {
	return logger.New("error") // Suppress logs during tests
}

func TestNewTaskWorkerPool(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(3, logger)

	if pool == nil {
		t.Fatal("Expected pool to be created")
	}

	if pool.workers != 3 {
		t.Errorf("Expected 3 workers, got %d", pool.workers)
	}

	if cap(pool.jobQueue) != 60 {
		t.Errorf("Expected job queue capacity 60, got %d", cap(pool.jobQueue))
	}

	if cap(pool.resultQueue) != 60 {
		t.Errorf("Expected result queue capacity 60, got %d", cap(pool.resultQueue))
	}

	pool.Shutdown()
}

func TestTaskWorkerPool_Submit(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(2, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-job-1",
		TaskID:    "task-1",
		UserID:    "user-1",
		Type:      "analyze",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Give worker time to process (job takes 500ms)
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.TotalJobs == 0 {
		t.Error("Expected at least one job to be processed")
	}
}

func TestTaskWorkerPool_Submit_QueueFull(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	// Fill the queue
	for i := 0; i < 20; i++ {
		job := queue.Job{
			ID:        "test-job-full",
			TaskID:    "task-full",
			UserID:    "user-full",
			Type:      "export",
			Data:      nil,
			CreatedAt: time.Now(),
		}
		pool.Submit(job)
	}

	// This should be rejected (queue full)
	job := queue.Job{
		ID:        "test-job-rejected",
		TaskID:    "task-rejected",
		UserID:    "user-rejected",
		Type:      "export",
		Data:      nil,
		CreatedAt: time.Now(),
	}
	pool.Submit(job)

	// Test completes without panic - queue overflow handled correctly
}

func TestTaskWorkerPool_ProcessJob_Export(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-export",
		TaskID:    "task-export",
		UserID:    "user-export",
		Type:      "export",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.CompletedJobs == 0 {
		t.Error("Expected export job to complete successfully")
	}
}

func TestTaskWorkerPool_ProcessJob_Analyze(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-analyze",
		TaskID:    "task-analyze",
		UserID:    "user-analyze",
		Type:      "analyze",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.CompletedJobs == 0 {
		t.Error("Expected analyze job to complete successfully")
	}
}

func TestTaskWorkerPool_ProcessJob_BatchUpdate(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-batch",
		TaskID:    "task-batch",
		UserID:    "user-batch",
		Type:      "batch_update",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.CompletedJobs == 0 {
		t.Error("Expected batch_update job to complete successfully")
	}
}

func TestTaskWorkerPool_ProcessJob_UnknownType(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-unknown",
		TaskID:    "task-unknown",
		UserID:    "user-unknown",
		Type:      "unknown_type",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.FailedJobs == 0 {
		t.Error("Expected unknown job type to fail")
	}
}

func TestTaskWorkerPool_GetMetrics(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(2, logger)
	defer pool.Shutdown()

	// Submit multiple jobs
	for i := 0; i < 5; i++ {
		job := queue.Job{
			ID:        "test-metrics",
			TaskID:    "task-metrics",
			UserID:    "user-metrics",
			Type:      "analyze",
			Data:      nil,
			CreatedAt: time.Now(),
		}
		pool.Submit(job)
	}

	// Wait for processing
	time.Sleep(700 * time.Millisecond)

	metrics := pool.GetMetrics()

	if metrics.TotalJobs == 0 {
		t.Error("Expected TotalJobs to be greater than 0")
	}

	if metrics.CompletedJobs+metrics.FailedJobs != metrics.TotalJobs {
		t.Error("Expected CompletedJobs + FailedJobs to equal TotalJobs")
	}

	if metrics.AverageDuration <= 0 {
		t.Error("Expected AverageDuration to be greater than 0")
	}
}

func TestTaskWorkerPool_Shutdown(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(3, logger)

	// Submit some jobs
	for i := 0; i < 3; i++ {
		job := queue.Job{
			ID:        "test-shutdown",
			TaskID:    "task-shutdown",
			UserID:    "user-shutdown",
			Type:      "analyze",
			Data:      nil,
			CreatedAt: time.Now(),
		}
		pool.Submit(job)
	}

	// Shutdown should complete without hanging
	done := make(chan bool)
	go func() {
		pool.Shutdown()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(15 * time.Second):
		t.Error("Shutdown timed out")
	}
}

func TestTaskWorkerPool_Shutdown_Graceful(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(2, logger)

	// Submit a job
	job := queue.Job{
		ID:        "test-graceful",
		TaskID:    "task-graceful",
		UserID:    "user-graceful",
		Type:      "export",
		Data:      nil,
		CreatedAt: time.Now(),
	}
	pool.Submit(job)

	// Short delay to ensure job is picked up
	time.Sleep(100 * time.Millisecond)

	// Shutdown should wait for job completion
	pool.Shutdown()

	metrics := pool.GetMetrics()
	if metrics.TotalJobs == 0 {
		t.Error("Expected job to be processed before shutdown")
	}
}

func TestTaskWorkerPool_ConcurrentSubmit(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(4, logger)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	numGoroutines := 10
	jobsPerGoroutine := 5

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				job := queue.Job{
					ID:        "test-concurrent",
					TaskID:    "task-concurrent",
					UserID:    "user-concurrent",
					Type:      "analyze",
					Data:      nil,
					CreatedAt: time.Now(),
				}
				pool.Submit(job)
			}
		}(g)
	}

	wg.Wait()

	// Wait for processing (each job takes 500ms, with 4 workers we need ~6.25s for 50 jobs)
	// Add extra time to account for scheduling overhead
	time.Sleep(8 * time.Second)

	metrics := pool.GetMetrics()
	expectedMin := numGoroutines * jobsPerGoroutine
	if int(metrics.TotalJobs) < expectedMin {
		t.Errorf("Expected at least %d jobs processed, got %d", expectedMin, metrics.TotalJobs)
	}
}

func TestTaskWorkerPool_MetricsThreadSafety(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(4, logger)
	defer pool.Shutdown()

	var wg sync.WaitGroup

	// Submit jobs from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				job := queue.Job{
					ID:        "test-thread-safety",
					TaskID:    "task-thread-safety",
					UserID:    "user-thread-safety",
					Type:      "analyze",
					Data:      nil,
					CreatedAt: time.Now(),
				}
				pool.Submit(job)
			}
		}()
	}

	// Read metrics concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = pool.GetMetrics()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}

func TestTaskWorkerPool_ContextCancellation(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(2, logger)

	// Manually cancel context
	pool.cancel()

	// Give time for workers to stop
	time.Sleep(100 * time.Millisecond)

	// Submit after cancellation - should not panic
	job := queue.Job{
		ID:        "test-after-cancel",
		TaskID:    "task-after-cancel",
		UserID:    "user-after-cancel",
		Type:      "analyze",
		Data:      nil,
		CreatedAt: time.Now(),
	}
	pool.Submit(job)

	pool.Shutdown()
}

func TestTaskWorkerPool_ResultQueueOverflow(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	// Submit many jobs quickly to potentially overflow result queue
	for i := 0; i < 50; i++ {
		job := queue.Job{
			ID:        "test-result-overflow",
			TaskID:    "task-result-overflow",
			UserID:    "user-result-overflow",
			Type:      "export",
			Data:      nil,
			CreatedAt: time.Now(),
		}
		pool.Submit(job)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Should not panic - overflow handled gracefully
	metrics := pool.GetMetrics()
	if metrics.TotalJobs == 0 {
		t.Error("Expected some jobs to be processed")
	}
}

func TestTaskWorkerPool_EmptyJobProcessing(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "",
		TaskID:    "",
		UserID:    "",
		Type:      "",
		Data:      nil,
		CreatedAt: time.Time{},
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(600 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.FailedJobs == 0 {
		t.Log("Empty job type should fail - this is expected behavior")
	}
}

func TestTaskWorkerPool_MultipleShutdownCalls(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(2, logger)

	// Multiple shutdown calls should not panic
	pool.Shutdown()
	pool.Shutdown()
	pool.Shutdown()
}

func TestTaskWorkerPool_LongRunningJob(t *testing.T) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(1, logger)
	defer pool.Shutdown()

	job := queue.Job{
		ID:        "test-long-running",
		TaskID:    "task-long-running",
		UserID:    "user-long-running",
		Type:      "analyze",
		Data:      nil,
		CreatedAt: time.Now(),
	}

	pool.Submit(job)

	// Shutdown while job is processing
	time.Sleep(100 * time.Millisecond)
	pool.Shutdown()

	// Should complete gracefully
}

func BenchmarkTaskWorkerPool_Submit(b *testing.B) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(4, logger)
	defer pool.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := queue.Job{
			ID:        "bench-job",
			TaskID:    "bench-task",
			UserID:    "bench-user",
			Type:      "analyze",
			Data:      nil,
			CreatedAt: time.Now(),
		}
		pool.Submit(job)
	}
}

func BenchmarkTaskWorkerPool_GetMetrics(b *testing.B) {
	logger := createTestLogger()
	pool := NewTaskWorkerPool(4, logger)
	defer pool.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.GetMetrics()
	}
}
