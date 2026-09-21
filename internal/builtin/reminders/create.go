package reminders

import (
	"github.com/vesvai/vesvai/internal/builtin/reminders/tasks"
	"github.com/vesvai/vesvai/internal/builtin/reminders/todowrite"
	"github.com/vesvai/vesvai/internal/builtin/reminders/usage"
	"github.com/vesvai/vesvai/internal/core/event"
)

func Create(bus event.Bus) error {
	if bus == nil {
		return nil
	}
	if err := usage.New().Start(bus); err != nil {
		return err
	}
	if err := todowrite.New().Start(bus); err != nil {
		return err
	}
	if err := tasks.New().Start(bus); err != nil {
		return err
	}
	return nil
}
