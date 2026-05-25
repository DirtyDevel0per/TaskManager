package worker

import (
	"testing"
	"time"

	"task-manager/internal/models"

	"github.com/google/uuid"
)

func TestBatchProcessor_Creation(t *testing.T) {
	logger := createTestLogger()
	
	bp := NewBatchProcessor(nil, 10, 100, 1*time.Second, logger)
	if bp == nil {
		t.Fatal("Expected BatchProcessor to be created")
	}
	
	bp.Shutdown()
}

func TestBatchProcessor_AddTask_BufferFull(t *testing.T) {
	logger := createTestLogger()
	
	// Create processor with very small buffer and long flush interval to avoid actual processing
	bp := NewBatchProcessor(nil, 10, 2, 1*time.Hour, logger)
	defer bp.Shutdown()
	
	// Just test that AddTask doesn't panic when buffer is full
	// Add tasks up to buffer capacity
	bp.AddTask(&models.Task{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Title: "task1"})
	bp.AddTask(&models.Task{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Title: "task2"})
	
	// This should be rejected gracefully (buffer full) - no panic
	bp.AddTask(&models.Task{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Title: "task3"})
	
	// Give time for potential flush attempt
	time.Sleep(10 * time.Millisecond)
	
	// Test completes without panic - buffer overflow handled correctly
}

func TestBatchProcessor_MultipleShutdown(t *testing.T) {
	logger := createTestLogger()
	
	bp := NewBatchProcessor(nil, 10, 100, 1*time.Second, logger)
	
	// Multiple shutdown calls should not panic
	bp.Shutdown()
	bp.Shutdown()
	bp.Shutdown()
}
