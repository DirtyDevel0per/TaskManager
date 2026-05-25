package worker

import (
	"testing"
	"time"

	"task-manager/internal/queue"
)

type mockNotifier struct{}

func (m *mockNotifier) Send(notification queue.Notification) {}
func (m *mockNotifier) NotifyTaskCreated(userID, taskID, taskTitle string) {}
func (m *mockNotifier) NotifyTaskOverdue(userID, taskID, taskTitle string) {}
func (m *mockNotifier) Shutdown() {}

func TestDeadlineChecker_Creation(t *testing.T) {
	logger := createTestLogger()
	
	dc := NewDeadlineChecker(nil, &mockNotifier{}, 1*time.Second, logger)
	if dc == nil {
		t.Fatal("Expected DeadlineChecker to be created")
	}
}

func TestDeadlineChecker_StartStop(t *testing.T) {
	logger := createTestLogger()
	
	dc := NewDeadlineChecker(nil, &mockNotifier{}, 100*time.Millisecond, logger)
	
	dc.Start()
	
	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)
	
	dc.Stop()
	
	// Should complete without hanging
}

func TestDeadlineChecker_MultipleStart(t *testing.T) {
	logger := createTestLogger()
	
	dc := NewDeadlineChecker(nil, &mockNotifier{}, 1*time.Second, logger)
	
	// Multiple starts should not cause issues
	dc.Start()
	dc.Start()
	dc.Start()
	
	dc.Stop()
}

func TestDeadlineChecker_MultipleStop(t *testing.T) {
	logger := createTestLogger()
	
	dc := NewDeadlineChecker(nil, &mockNotifier{}, 1*time.Second, logger)
	
	dc.Start()
	time.Sleep(50 * time.Millisecond)
	
	// Multiple stops should not panic
	dc.Stop()
	dc.Stop()
	dc.Stop()
}
