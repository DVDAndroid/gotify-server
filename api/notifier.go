package api

import "github.com/gotify/server/v2/model"

// Notifier notifies when a new message was created.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}
