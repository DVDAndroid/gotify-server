package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

// GetMessageByID returns the messages for the given id or nil.
func (d *GormDatabase) GetMessageByID(id uint) (*model.Message, error) {
	msg := new(model.Message)
	err := d.DB.Find(msg, id).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	if msg.ID == id {
		return msg, err
	}
	return nil, err
}

// CreateMessage creates a message.
func (d *GormDatabase) CreateMessage(message *model.Message) error {
	return d.DB.Create(message).Error
}

// GetMessagesByUser returns all messages from a user.
func (d *GormDatabase) GetMessagesByUser(userID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Joins("JOIN applications ON applications.user_id = ?", userID).
		Where("messages.application_id = applications.id").Order("messages.id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByUserSince returns limited messages from a user.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByUserSince(userID uint, limit int, since uint, postponed *string) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.Joins("JOIN applications ON applications.user_id = ?", userID).
		Where("messages.application_id = applications.id").Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := filterPostponed(db, postponed).Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplication returns all messages from an application.
func (d *GormDatabase) GetMessagesByApplication(tokenID uint) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Where("application_id = ?", tokenID).Order("messages.id desc").Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// GetMessagesByApplicationSince returns limited messages from an application.
// If since is 0 it will be ignored.
func (d *GormDatabase) GetMessagesByApplicationSince(appID uint, limit int, since uint, postponed *string) ([]*model.Message, error) {
	var messages []*model.Message
	db := d.DB.Where("application_id = ?", appID).Order("messages.id desc").Limit(limit)
	if since != 0 {
		db = db.Where("messages.id < ?", since)
	}
	err := filterPostponed(db, postponed).Find(&messages).Error
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	return messages, err
}

// DeleteMessageByID deletes a message by its id.
func (d *GormDatabase) DeleteMessageByID(id uint) error {
	return d.DB.Where("id = ?", id).Delete(&model.Message{}).Error
}

// DeleteMessagesByApplication deletes all messages from an application.
func (d *GormDatabase) DeleteMessagesByApplication(applicationID uint) error {
	return d.DB.Where("application_id = ?", applicationID).Delete(&model.Message{}).Error
}

// DeleteMessagesByUser deletes all messages from a user.
func (d *GormDatabase) DeleteMessagesByUser(userID uint) error {
	app, _ := d.GetApplicationsByUser(userID)
	for _, app := range app {
		d.DeleteMessagesByApplication(app.ID)
	}
	return nil
}

// filterPostponed restricts the query by postponement: "all" or nil disables the filter,
// "true" (or empty) keeps only postponed messages, anything else only not postponed ones.
func filterPostponed(db *gorm.DB, postponed *string) *gorm.DB {
	switch {
	case postponed == nil || *postponed == "all":
		return db
	case *postponed == "true" || *postponed == "":
		return db.Where("messages.postponed_at IS NOT NULL")
	default:
		return db.Where("messages.postponed_at IS NULL")
	}
}

// UpdateMessagePostponement sets (or clears, if nil) the date a message is postponed to
// and marks the message as not delivered yet.
func (d *GormDatabase) UpdateMessagePostponement(id uint, postponedAt *time.Time) error {
	return d.DB.Model(&model.Message{}).Where("id = ?", id).
		Updates(map[string]any{"postponed_at": postponedAt, "postpone_pending": postponedAt != nil}).Error
}

// MarkMessagePostponementDelivered records that a postponed message was pushed to the clients.
func (d *GormDatabase) MarkMessagePostponementDelivered(id uint) error {
	return d.DB.Model(&model.Message{}).Where("id = ?", id).Update("postpone_pending", false).Error
}

// GetMessagesToDeliver returns the postponed messages which still have to be pushed: the ones
// postponed to a date after now and the ones missed (pending, but their date has passed).
// Messages postponed before the pending flag existed are only found by their future date.
func (d *GormDatabase) GetMessagesToDeliver(now time.Time) ([]*model.Message, error) {
	var messages []*model.Message
	err := d.DB.Where("postponed_at IS NOT NULL AND (postponed_at > ? OR postpone_pending = ?)", now, true).Find(&messages).Error
	return messages, err
}
