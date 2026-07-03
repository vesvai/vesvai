package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
)

func newTestBus() event.EventBus {
	return event.NewEventBus(event.EventBusConfig{
		EnableDeadEvent: false,
	})
}

func TestNewManager_DefaultConfig(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus)

	if manager.config.AppName != "Vesvai" {
		t.Errorf("AppName = %q, want %q", manager.config.AppName, "Vesvai")
	}
	if !manager.config.EnableOS {
		t.Error("EnableOS = false, want true")
	}
	if manager.config.MaxNotifications != 1000 {
		t.Errorf("MaxNotifications = %d, want 1000", manager.config.MaxNotifications)
	}
}

func TestNewManager_CustomConfig(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	config := ManagerConfig{
		AppName:          "TestApp",
		EnableOS:         false,
		MaxNotifications: 100,
	}
	manager := NewManager(bus, config)

	if manager.config.AppName != "TestApp" {
		t.Errorf("AppName = %q, want %q", manager.config.AppName, "TestApp")
	}
	if manager.config.EnableOS {
		t.Error("EnableOS = true, want false")
	}
	if manager.config.MaxNotifications != 100 {
		t.Errorf("MaxNotifications = %d, want 100", manager.config.MaxNotifications)
	}
}

func TestManager_Publish(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	notif, err := manager.Publish(
		context.Background(),
		"Test Title",
		"Test Message",
		TypeInfo,
		PriorityNormal,
	)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if notif.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", notif.Title, "Test Title")
	}
	if notif.Message != "Test Message" {
		t.Errorf("Message = %q, want %q", notif.Message, "Test Message")
	}
	if notif.Type != TypeInfo {
		t.Errorf("Type = %q, want %q", notif.Type, TypeInfo)
	}
	if notif.Priority != PriorityNormal {
		t.Errorf("Priority = %d, want %d", notif.Priority, PriorityNormal)
	}
	if notif.Read {
		t.Error("Read = true, want false")
	}
	if notif.ID == "" {
		t.Error("ID is empty")
	}
}

func TestManager_Publish_EventPublished(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	var received *NotificationEvent
	bus.Subscribe(
		event.EventType(EventNotificationPublished),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			received = e.(*NotificationEvent)
			return nil
		}),
	)

	manager.Publish(
		context.Background(),
		"Event Test",
		"Event Message",
		TypeSuccess,
		PriorityHigh,
	)

	if received == nil {
		t.Fatal("notification event not received")
	}
	if received.Notification.Title != "Event Test" {
		t.Errorf("event Title = %q, want %q", received.Notification.Title, "Event Test")
	}
}

func TestManager_PublishAsync(t *testing.T) {
	bus := event.NewEventBus(event.EventBusConfig{
		EnableDeadEvent: false,
		AsyncQueueSize:  1024,
	})
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(
		event.EventType(EventNotificationPublished),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			defer wg.Done()
			return nil
		}),
	)

	// Small delay to ensure subscriber is ready
	time.Sleep(10 * time.Millisecond)

	err := manager.PublishAsync(
		context.Background(),
		"Async Test",
		"Async Message",
		TypeInfo,
		PriorityNormal,
	)
	if err != nil {
		t.Fatalf("PublishAsync() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async event")
	}
}

func TestManager_MarkAsRead(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	notif, _ := manager.Publish(
		context.Background(),
		"Read Test",
		"Read Message",
		TypeInfo,
		PriorityNormal,
	)

	var readEvent *NotificationEvent
	bus.Subscribe(
		event.EventType(EventNotificationRead),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			readEvent = e.(*NotificationEvent)
			return nil
		}),
	)

	err := manager.MarkAsRead(context.Background(), notif.ID)
	if err != nil {
		t.Fatalf("MarkAsRead() error = %v", err)
	}

	if !notif.Read {
		t.Error("Read = false, want true")
	}
	if notif.ReadAt == nil {
		t.Error("ReadAt is nil")
	}
	if readEvent == nil {
		t.Error("read event not published")
	}
}

func TestManager_MarkAsRead_NotFound(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	err := manager.MarkAsRead(context.Background(), "nonexistent")
	if err == nil {
		t.Error("MarkAsRead() on nonexistent should error")
	}
}

func TestManager_Dismiss(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	notif, _ := manager.Publish(
		context.Background(),
		"Dismiss Test",
		"Dismiss Message",
		TypeInfo,
		PriorityNormal,
	)

	var dismissEvent *NotificationEvent
	bus.Subscribe(
		event.EventType(EventNotificationDismissed),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			dismissEvent = e.(*NotificationEvent)
			return nil
		}),
	)

	err := manager.Dismiss(context.Background(), notif.ID)
	if err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Count = %d, want 0 after dismiss", manager.Count())
	}
	if dismissEvent == nil {
		t.Error("dismiss event not published")
	}
}

func TestManager_Dismiss_NotFound(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	err := manager.Dismiss(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Dismiss() on nonexistent should error")
	}
}

func TestManager_Clear(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	manager.Publish(context.Background(), "Test 1", "Message 1", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Test 2", "Message 2", TypeInfo, PriorityNormal)

	var clearEvent *NotificationEvent
	bus.Subscribe(
		event.EventType(EventNotificationCleared),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			clearEvent = e.(*NotificationEvent)
			return nil
		}),
	)

	err := manager.Clear(context.Background())
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Count = %d, want 0 after clear", manager.Count())
	}
	if clearEvent == nil {
		t.Error("clear event not published")
	}
}

func TestManager_GetAll(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	manager.Publish(context.Background(), "Test 1", "Message 1", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Test 2", "Message 2", TypeSuccess, PriorityHigh)

	all := manager.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() len = %d, want 2", len(all))
	}
}

func TestManager_GetUnread(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	notif1, _ := manager.Publish(context.Background(), "Test 1", "Message 1", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Test 2", "Message 2", TypeInfo, PriorityNormal)

	manager.MarkAsRead(context.Background(), notif1.ID)

	unread := manager.GetUnread()
	if len(unread) != 1 {
		t.Errorf("GetUnread() len = %d, want 1", len(unread))
	}
}

func TestManager_GetFiltered(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	manager.Publish(context.Background(), "Info", "Info msg", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Error", "Error msg", TypeError, PriorityUrgent)
	manager.Publish(context.Background(), "Warning", "Warning msg", TypeWarning, PriorityHigh)

	// Filter by type
	filtered := manager.GetFiltered(NotificationFilter{
		Types: []NotificationType{TypeError},
	})
	if len(filtered) != 1 {
		t.Errorf("GetFiltered by type len = %d, want 1", len(filtered))
	}

	// Filter by priority
	filtered = manager.GetFiltered(NotificationFilter{
		MinPriority: PriorityHigh,
	})
	if len(filtered) != 2 {
		t.Errorf("GetFiltered by priority len = %d, want 2", len(filtered))
	}

	// Filter with limit
	filtered = manager.GetFiltered(NotificationFilter{
		Limit: 1,
	})
	if len(filtered) != 1 {
		t.Errorf("GetFiltered with limit len = %d, want 1", len(filtered))
	}
}

func TestManager_Count(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	if manager.Count() != 0 {
		t.Errorf("Count = %d, want 0 initially", manager.Count())
	}

	manager.Publish(context.Background(), "Test 1", "Message 1", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Test 2", "Message 2", TypeInfo, PriorityNormal)

	if manager.Count() != 2 {
		t.Errorf("Count = %d, want 2", manager.Count())
	}
}

func TestManager_UnreadCount(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	notif1, _ := manager.Publish(context.Background(), "Test 1", "Message 1", TypeInfo, PriorityNormal)
	manager.Publish(context.Background(), "Test 2", "Message 2", TypeInfo, PriorityNormal)

	if manager.UnreadCount() != 2 {
		t.Errorf("UnreadCount = %d, want 2", manager.UnreadCount())
	}

	manager.MarkAsRead(context.Background(), notif1.ID)

	if manager.UnreadCount() != 1 {
		t.Errorf("UnreadCount = %d, want 1 after read", manager.UnreadCount())
	}
}

func TestManager_MaxNotifications(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{
		EnableOS:         false,
		MaxNotifications: 3,
	})

	for i := 0; i < 5; i++ {
		manager.Publish(
			context.Background(),
			"Test",
			"Message",
			TypeInfo,
			PriorityNormal,
		)
	}

	if manager.Count() != 3 {
		t.Errorf("Count = %d, want 3 (max)", manager.Count())
	}
}

func TestFilterByNotificationType(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	filter := FilterByNotificationType(TypeError, TypeWarning)

	// Should pass
	errorNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Type: TypeError})
	if !filter(errorNotif) {
		t.Error("Filter should pass for TypeError")
	}

	// Should pass
	warningNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Type: TypeWarning})
	if !filter(warningNotif) {
		t.Error("Filter should pass for TypeWarning")
	}

	// Should fail
	infoNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Type: TypeInfo})
	if filter(infoNotif) {
		t.Error("Filter should fail for TypeInfo")
	}

	_ = manager
}

func TestFilterByMinPriority(t *testing.T) {
	filter := FilterByMinPriority(PriorityHigh)

	// Should pass
	highNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Priority: PriorityHigh})
	if !filter(highNotif) {
		t.Error("Filter should pass for PriorityHigh")
	}

	urgentNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Priority: PriorityUrgent})
	if !filter(urgentNotif) {
		t.Error("Filter should pass for PriorityUrgent")
	}

	// Should fail
	normalNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Priority: PriorityNormal})
	if filter(normalNotif) {
		t.Error("Filter should fail for PriorityNormal")
	}
}

func TestFilterByUnread(t *testing.T) {
	filter := FilterByUnread()

	// Should pass
	unreadNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Read: false})
	if !filter(unreadNotif) {
		t.Error("Filter should pass for unread")
	}

	// Should fail
	readNotif := NewNotificationEvent(EventNotificationPublished, &Notification{Read: true})
	if filter(readNotif) {
		t.Error("Filter should fail for read")
	}
}

func TestHelperFunctions(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})
	ctx := context.Background()

	// Test Info
	notif, err := Info(ctx, manager, "Info Title", "Info Message")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if notif.Type != TypeInfo {
		t.Errorf("Info() Type = %q, want %q", notif.Type, TypeInfo)
	}

	// Test Success
	notif, err = Success(ctx, manager, "Success Title", "Success Message")
	if err != nil {
		t.Fatalf("Success() error = %v", err)
	}
	if notif.Type != TypeSuccess {
		t.Errorf("Success() Type = %q, want %q", notif.Type, TypeSuccess)
	}

	// Test Warning
	notif, err = Warning(ctx, manager, "Warning Title", "Warning Message")
	if err != nil {
		t.Fatalf("Warning() error = %v", err)
	}
	if notif.Type != TypeWarning {
		t.Errorf("Warning() Type = %q, want %q", notif.Type, TypeWarning)
	}
	if notif.Priority != PriorityHigh {
		t.Errorf("Warning() Priority = %d, want %d", notif.Priority, PriorityHigh)
	}

	// Test Error
	notif, err = Error(ctx, manager, "Error Title", "Error Message")
	if err != nil {
		t.Fatalf("Error() error = %v", err)
	}
	if notif.Type != TypeError {
		t.Errorf("Error() Type = %q, want %q", notif.Type, TypeError)
	}
	if notif.Priority != PriorityUrgent {
		t.Errorf("Error() Priority = %d, want %d", notif.Priority, PriorityUrgent)
	}
}

func TestNotificationEvent_Type(t *testing.T) {
	notif := &Notification{
		ID:   "test-id",
		Type: TypeInfo,
	}

	notifEvent := NewNotificationEvent(EventNotificationPublished, notif)
	if notifEvent.Type() != event.EventType(EventNotificationPublished) {
		t.Errorf("Type() = %q, want %q", notifEvent.Type(), EventNotificationPublished)
	}
}

func TestNotificationEvent_Timestamp(t *testing.T) {
	before := time.Now()
	notif := &Notification{ID: "test-id"}
	event := NewNotificationEvent(EventNotificationPublished, notif)
	after := time.Now()

	if event.Timestamp().Before(before) || event.Timestamp().After(after) {
		t.Errorf("Timestamp() = %v, not between %v and %v", event.Timestamp(), before, after)
	}
}

func TestSubscribeToNotifications(t *testing.T) {
	bus := newTestBus()
	defer bus.Close()

	manager := NewManager(bus, ManagerConfig{EnableOS: false})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := manager.SubscribeToNotifications(ctx)
	if err != nil {
		t.Fatalf("SubscribeToNotifications() error = %v", err)
	}

	// Publish a notification
	go func() {
		manager.Publish(
			context.Background(),
			"Sub Test",
			"Sub Message",
			TypeInfo,
			PriorityNormal,
		)
	}()

	// Should receive the notification
	select {
	case notif := <-ch:
		if notif.Title != "Sub Test" {
			t.Errorf("received Title = %q, want %q", notif.Title, "Sub Test")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for notification")
	}
}
