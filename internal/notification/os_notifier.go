package notification

import (
	"fmt"
	"log"

	"github.com/gen2brain/beeep"
)

type OSNotifier struct {
	appName string
}

func NewOSNotifier(appName string) *OSNotifier {
	return &OSNotifier{
		appName: appName,
	}
}

func (n *OSNotifier) Send(title, message string, notifType NotificationType) error {
	icon := n.getIcon(notifType)

	err := beeep.Notify(title, message, icon)
	if err != nil {
		return fmt.Errorf("failed to send OS notification: %w", err)
	}

	log.Printf("OS notification sent: %s - %s", title, message)
	return nil
}

func (n *OSNotifier) SendWithTitle(title, message string, notifType NotificationType) error {
	fullTitle := fmt.Sprintf("[%s] %s", n.appName, title)
	return n.Send(fullTitle, message, notifType)
}

func (n *OSNotifier) getIcon(notifType NotificationType) string {
	return ""
}

func (n *OSNotifier) Beep() error {
	return beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
}

func (n *OSNotifier) Alert(title, message string) error {
	if err := n.Beep(); err != nil {
		log.Printf("failed to beep: %v", err)
	}
	return n.Send(title, message, TypeError)
}
