package notification

import (
	"time"

	"github.com/vesvai/vesvai/internal/event"
)

type NotificationEventType event.EventType

const (
	EventNotificationPublished NotificationEventType = "notification:published"

	EventNotificationRead NotificationEventType = "notification:read"

	EventNotificationDismissed NotificationEventType = "notification:dismissed"

	EventNotificationCleared NotificationEventType = "notification:cleared"

	EventNotificationError NotificationEventType = "notification:error"
)

type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 50
	PriorityHigh   Priority = 100
	PriorityUrgent Priority = 150
)

type NotificationType string

const (
	TypeInfo    NotificationType = "info"
	TypeSuccess NotificationType = "success"
	TypeWarning NotificationType = "warning"
	TypeError   NotificationType = "error"
)

type Notification struct {
	ID        string
	Title     string
	Message   string
	Type      NotificationType
	Priority  Priority
	Read      bool
	CreatedAt time.Time
	ReadAt    *time.Time
	Metadata  map[string]string
}

type NotificationEvent struct {
	event.BaseEvent
	Notification *Notification
}

func NewNotificationEvent(eventType NotificationEventType, notification *Notification) *NotificationEvent {
	return &NotificationEvent{
		BaseEvent:    event.NewBaseEvent(event.EventType(eventType)),
		Notification: notification,
	}
}

type NotificationEventData struct {
	NotificationID string
	Title          string
	Message        string
	Type           NotificationType
	Priority       Priority
	Read           bool
}

type NotificationError struct {
	NotificationID string
	Error          error
	Context        string
}

type NotificationFilter struct {
	Types       []NotificationType
	MinPriority Priority
	ReadStatus  *bool
	Since       *time.Time
	Limit       int
}
