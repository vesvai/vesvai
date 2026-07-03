package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vesvai/vesvai/internal/event"
)

type Manager struct {
	bus           event.EventBus
	osNotifier    *OSNotifier
	notifications []*Notification
	mu            sync.RWMutex
	config        ManagerConfig
}

type ManagerConfig struct {
	AppName          string
	EnableOS         bool
	MaxNotifications int
}

func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		AppName:          "Vesvai",
		EnableOS:         true,
		MaxNotifications: 1000,
	}
}

func NewManager(bus event.EventBus, configs ...ManagerConfig) *Manager {
	config := DefaultManagerConfig()
	if len(configs) > 0 {
		config = configs[0]
	}

	return &Manager{
		bus:           bus,
		osNotifier:    NewOSNotifier(config.AppName),
		notifications: make([]*Notification, 0),
		config:        config,
	}
}

func (m *Manager) Publish(ctx context.Context, title, message string, notifType NotificationType, priority Priority) (*Notification, error) {
	notif := &Notification{
		ID:        uuid.New().String(),
		Title:     title,
		Message:   message,
		Type:      notifType,
		Priority:  priority,
		Read:      false,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}

	m.mu.Lock()
	m.notifications = append(m.notifications, notif)
	m.trimNotifications()
	m.mu.Unlock()

	if m.config.EnableOS {
		if err := m.osNotifier.SendWithTitle(title, message, notifType); err != nil {
			log.Printf("failed to send OS notification: %v", err)
		}
	}

	event := NewNotificationEvent(EventNotificationPublished, notif)
	if err := m.bus.Publish(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to publish notification event: %w", err)
	}

	log.Printf("notification published: %s - %s", title, message)
	return notif, nil
}

func (m *Manager) PublishAsync(ctx context.Context, title, message string, notifType NotificationType, priority Priority) error {
	notif := &Notification{
		ID:        uuid.New().String(),
		Title:     title,
		Message:   message,
		Type:      notifType,
		Priority:  priority,
		Read:      false,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}

	m.mu.Lock()
	m.notifications = append(m.notifications, notif)
	m.trimNotifications()
	m.mu.Unlock()

	if m.config.EnableOS {
		go func() {
			if err := m.osNotifier.SendWithTitle(title, message, notifType); err != nil {
				log.Printf("failed to send OS notification: %v", err)
			}
		}()
	}

	event := NewNotificationEvent(EventNotificationPublished, notif)
	return m.bus.PublishAsync(ctx, event)
}

func (m *Manager) MarkAsRead(ctx context.Context, notificationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, notif := range m.notifications {
		if notif.ID == notificationID {
			notif.Read = true
			now := time.Now()
			notif.ReadAt = &now

			// Publish read event
			event := NewNotificationEvent(EventNotificationRead, notif)
			return m.bus.Publish(ctx, event)
		}
	}

	return fmt.Errorf("notification not found: %s", notificationID)
}

func (m *Manager) Dismiss(ctx context.Context, notificationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, notif := range m.notifications {
		if notif.ID == notificationID {
			m.notifications = append(m.notifications[:i], m.notifications[i+1:]...)

			event := NewNotificationEvent(EventNotificationDismissed, notif)
			return m.bus.Publish(ctx, event)
		}
	}

	return fmt.Errorf("notification not found: %s", notificationID)
}

func (m *Manager) Clear(ctx context.Context) error {
	m.mu.Lock()
	m.notifications = make([]*Notification, 0)
	m.mu.Unlock()

	event := NewNotificationEvent(EventNotificationCleared, nil)
	return m.bus.Publish(ctx, event)
}

func (m *Manager) GetAll() []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *Manager) GetUnread() []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Notification, 0)
	for _, notif := range m.notifications {
		if !notif.Read {
			result = append(result, notif)
		}
	}
	return result
}

func (m *Manager) GetFiltered(filter NotificationFilter) []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Notification, 0)
	for _, notif := range m.notifications {
		if m.matchesFilter(notif, filter) {
			result = append(result, notif)
		}
	}

	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.notifications)
}

func (m *Manager) UnreadCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, notif := range m.notifications {
		if !notif.Read {
			count++
		}
	}
	return count
}

func (m *Manager) matchesFilter(notif *Notification, filter NotificationFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if notif.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if notif.Priority < filter.MinPriority {
		return false
	}

	if filter.ReadStatus != nil {
		if *filter.ReadStatus != notif.Read {
			return false
		}
	}

	if filter.Since != nil {
		if notif.CreatedAt.Before(*filter.Since) {
			return false
		}
	}

	return true
}

func (m *Manager) trimNotifications() {
	if m.config.MaxNotifications <= 0 {
		return
	}

	for len(m.notifications) > m.config.MaxNotifications {
		m.notifications = m.notifications[1:]
	}
}

func (m *Manager) SubscribeToNotifications(ctx context.Context) (<-chan *Notification, error) {
	ch := make(chan *Notification, 100)

	handler := event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		notifEvent, ok := e.(*NotificationEvent)
		if !ok {
			return nil
		}

		if notifEvent.Notification != nil {
			select {
			case ch <- notifEvent.Notification:
			default:
				log.Printf("notification channel full, dropping: %s", notifEvent.Notification.Title)
			}
		}
		return nil
	})

	_, err := m.bus.Subscribe(
		event.EventType(EventNotificationPublished),
		handler,
		event.WithPriority(event.PriorityNormal),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to notifications: %w", err)
	}

	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}
