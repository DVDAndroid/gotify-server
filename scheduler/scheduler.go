package scheduler

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/gotify/server/v2/api/stream"
	"github.com/gotify/server/v2/database"
	"github.com/gotify/server/v2/model"
	"time"
)

type Scheduler struct {
	db        *database.GormDatabase
	scheduler gocron.Scheduler
	api       *stream.API
}

// map message id -> job id
var jobs = make(map[uint]uuid.UUID)

func Init(database *database.GormDatabase, api *stream.API) (Scheduler, func() error) {
	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		fmt.Println("scheduler error: ", err)
		panic(err)
	}

	cronScheduler.Start()
	scheduler := Scheduler{
		db:        database,
		api:       api,
		scheduler: cronScheduler,
	}
	scheduler.scheduleAll()

	return scheduler, cronScheduler.Shutdown
}

func (s Scheduler) scheduleAll() {
	var messages []*model.Message
	s.db.DB.Where("postponed_at >= ?", time.Now()).Find(&messages)
	for _, message := range messages {
		s.ScheduleMessage(message.ID, *message.PostponedAt)
	}
}

func (s Scheduler) ScheduleMessage(msgId uint, postponedAt time.Time) {
	job, err := s.scheduler.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(postponedAt),
		),
		gocron.NewTask(func() {
			var message model.Message
			db := s.db.DB.Where("id = ?", msgId).First(&message)
			if db.Error != nil {
				fmt.Println("Error getting message with id ", msgId, db.Error)
				panic(db.Error)
			}
			userId := message.ApplicationID
			// remove from the job list only, i want to keep track of the postponed date and time
			s.DeleteMessageSchedule(&message)
			s.api.Notify(userId, ToExternalMessage(&message))
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling message with id ", msgId, err)
		return
	}
	jobs[msgId] = job.ID()
}

func (s Scheduler) DeleteMessageSchedule(message *model.Message) {
	jobId, ok := jobs[message.ID]
	if !ok {
		return
	}
	s.scheduler.RemoveJob(jobId)
	delete(jobs, message.ID)
}

func (s Scheduler) DeleteMessagesScheduleByApplication(appID uint) {
	var messages []*model.Message
	s.db.DB.Where("application_id = ?", appID).Find(&messages)
	for _, message := range messages {
		s.DeleteMessageSchedule(message)
	}
}

func (s Scheduler) DeleteMessagesScheduleByUser(userID uint) {
	app, _ := s.db.GetApplicationsByUser(userID)
	for _, app := range app {
		s.DeleteMessagesScheduleByApplication(app.ID)
	}
}

// moved from api/message.go due to circular imports
func ToExternalMessage(msg *model.Message) *model.MessageExternal {
	res := &model.MessageExternal{
		ID:            msg.ID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Priority:      &msg.Priority,
		Date:          msg.Date,
		PostponedAt:   msg.PostponedAt,
	}
	if len(msg.Extras) != 0 {
		res.Extras = make(map[string]interface{})
		json.Unmarshal(msg.Extras, &res.Extras)
	}
	return res
}
