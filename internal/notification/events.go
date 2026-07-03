package notification

import (
	"context"
	"log"

	"github.com/vesvai/vesvai/internal/event"
)

func FilterByNotificationType(types ...NotificationType) event.FilterFunc {
	return func(e event.Event) bool {
		ne, ok := e.(*NotificationEvent)
		if !ok {
			return false
		}

		for _, t := range types {
			if ne.Notification.Type == t {
				return true
			}
		}
		return false
	}
}

func FilterByMinPriority(minPriority Priority) event.FilterFunc {
	return func(e event.Event) bool {
		ne, ok := e.(*NotificationEvent)
		if !ok {
			return false
		}
		return ne.Notification.Priority >= minPriority
	}
}

func FilterByUnread() event.FilterFunc {
	return func(e event.Event) bool {
		ne, ok := e.(*NotificationEvent)
		if !ok {
			return false
		}
		return !ne.Notification.Read
	}
}

func RegisterNotificationHandlers(bus event.EventBus, manager *Manager) error {
	_, err := bus.Subscribe(
		event.EventType(EventNotificationPublished),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			ne, ok := e.(*NotificationEvent)
			if !ok {
				return nil
			}
			log.Printf("notification published: [%s] %s - %s",
				ne.Notification.Type, ne.Notification.Title, ne.Notification.Message)
			return nil
		}),
		event.WithPriority(event.PriorityLow),
	)
	if err != nil {
		return err
	}

	_, err = bus.Subscribe(
		event.EventType(EventNotificationError),
		event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			ne, ok := e.(*NotificationEvent)
			if !ok {
				return nil
			}
			log.Printf("notification error for %s", ne.Notification.ID)
			return nil
		}),
		event.WithPriority(event.PriorityHigh),
	)
	if err != nil {
		return err
	}

	return nil
}

func Info(ctx context.Context, manager *Manager, title, message string) (*Notification, error) {
	return manager.Publish(ctx, title, message, TypeInfo, PriorityNormal)
}

func Success(ctx context.Context, manager *Manager, title, message string) (*Notification, error) {
	return manager.Publish(ctx, title, message, TypeSuccess, PriorityNormal)
}

func Warning(ctx context.Context, manager *Manager, title, message string) (*Notification, error) {
	return manager.Publish(ctx, title, message, TypeWarning, PriorityHigh)
}

func Error(ctx context.Context, manager *Manager, title, message string) (*Notification, error) {
	return manager.Publish(ctx, title, message, TypeError, PriorityUrgent)
}
