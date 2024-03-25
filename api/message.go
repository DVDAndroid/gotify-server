package api

import (
	"encoding/json"
	"errors"
	"github.com/gotify/server/v2/scheduler"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gotify/location"
	"github.com/gotify/server/v2/auth"
	"github.com/gotify/server/v2/model"
)

// The MessageDatabase interface for encapsulating database access.
type MessageDatabase interface {
	GetMessagesByApplicationSince(appID uint, limit int, since uint, postponed *string) ([]*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetMessagesByUserSince(userID uint, limit int, since uint, postponed *string) ([]*model.Message, error)
	DeleteMessageByID(id uint) error
	GetMessageByID(id uint) (*model.Message, error)
	DeleteMessagesByUser(userID uint) error
	DeleteMessagesByApplication(applicationID uint) error
	CreateMessage(message *model.Message) error
	GetApplicationByToken(token string) (*model.Application, error)
	UpdateMessagePostponement(id uint, postponedAt *time.Time) error
}

var timeNow = time.Now

// The MessageAPI provides handlers for managing messages.
type MessageAPI struct {
	DB        MessageDatabase
	Notifier  Notifier
	Scheduler scheduler.Scheduler
}

type pagingParams struct {
	Limit     int     `form:"limit" binding:"min=1,max=200"`
	Since     uint    `form:"since" binding:"min=0"`
	Postponed *string `form:"postponed"`
}

// GetMessages returns all messages from a user.
// swagger:operation GET /message message getMessages
//
// Return all messages.
//
//	---
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: limit
//	  in: query
//	  description: the maximal amount of messages to return
//	  required: false
//	  maximum: 200
//	  minimum: 1
//	  default: 100
//	  type: integer
//	- name: since
//	  in: query
//	  description: return all messages with an ID less than this value
//	  minimum: 0
//	  required: false
//	  type: integer
//	  format: int64
//	- name: postponed
//	  in: query
//	  description: return all, only postponed or only not postponed messages
//	  required: false
//	  type: string
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/PagedMessages"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) GetMessages(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	withPaging(ctx, func(params *pagingParams) {
		// the +1 is used to check if there are more messages and will be removed on buildWithPaging
		messages, err := a.DB.GetMessagesByUserSince(userID, params.Limit+1, params.Since, params.Postponed)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		ctx.JSON(200, buildWithPaging(ctx, params, messages))
	})
}

func buildWithPaging(ctx *gin.Context, paging *pagingParams, messages []*model.Message) *model.PagedMessages {
	next := ""
	since := uint(0)
	useMessages := messages
	if len(messages) > paging.Limit {
		useMessages = messages[:len(messages)-1]
		since = useMessages[len(useMessages)-1].ID
		url := location.Get(ctx)
		url.Path = ctx.Request.URL.Path
		query := url.Query()
		query.Add("limit", strconv.Itoa(paging.Limit))
		query.Add("since", strconv.FormatUint(uint64(since), 10))
		url.RawQuery = query.Encode()
		next = url.String()
	}
	return &model.PagedMessages{
		Paging:   model.Paging{Size: len(useMessages), Limit: paging.Limit, Next: next, Since: since},
		Messages: toExternalMessages(useMessages),
	}
}

func withPaging(ctx *gin.Context, f func(pagingParams *pagingParams)) {
	params := &pagingParams{Limit: 100}
	if err := ctx.MustBindWith(params, binding.Query); err == nil {
		f(params)
	}
}

// GetMessagesWithApplication returns all messages from a specific application.
// swagger:operation GET /application/{id}/message message getAppMessages
//
// Return all messages from a specific application.
//
//	---
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	- name: limit
//	  in: query
//	  description: the maximal amount of messages to return
//	  required: false
//	  maximum: 200
//	  minimum: 1
//	  default: 100
//	  type: integer
//	- name: since
//	  in: query
//	  description: return all messages with an ID less than this value
//	  minimum: 0
//	  required: false
//	  type: integer
//	  format: int64
//	- name: postponed
//	  in: query
//	  description: return all, only postponed or only not postponed messages
//	  required: false
//	  type: string
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	        $ref: "#/definitions/PagedMessages"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) GetMessagesWithApplication(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		withPaging(ctx, func(params *pagingParams) {
			app, err := a.DB.GetApplicationByID(id)
			if success := successOrAbort(ctx, 500, err); !success {
				return
			}
			if app != nil && app.UserID == auth.GetUserID(ctx) {
				// the +1 is used to check if there are more messages and will be removed on buildWithPaging
				messages, err := a.DB.GetMessagesByApplicationSince(id, params.Limit+1, params.Since, params.Postponed)
				if success := successOrAbort(ctx, 500, err); !success {
					return
				}
				ctx.JSON(200, buildWithPaging(ctx, params, messages))
			} else {
				ctx.AbortWithError(404, errors.New("application does not exist"))
			}
		})
	})
}

// DeleteMessages delete all messages from a user.
// swagger:operation DELETE /message message deleteMessages
//
// Delete all messages.
//
//	---
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	responses:
//	  200:
//	    description: Ok
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) DeleteMessages(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	a.Scheduler.DeleteMessagesScheduleByUser(userID)
	successOrAbort(ctx, 500, a.DB.DeleteMessagesByUser(userID))
}

// DeleteMessageWithApplication deletes all messages from a specific application.
// swagger:operation DELETE /application/{id}/message message deleteAppMessages
//
// Delete all messages from a specific application.
//
//	---
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: id
//	  in: path
//	  description: the application id
//	  required: true
//	  type: integer
//	  format: int64
//	responses:
//	  200:
//	    description: Ok
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) DeleteMessageWithApplication(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		application, err := a.DB.GetApplicationByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if application != nil && application.UserID == auth.GetUserID(ctx) {
			a.Scheduler.DeleteMessagesScheduleByApplication(id)
			successOrAbort(ctx, 500, a.DB.DeleteMessagesByApplication(id))
		} else {
			ctx.AbortWithError(404, errors.New("application does not exists"))
		}
	})
}

// DeleteMessage deletes a message with an id.
// swagger:operation DELETE /message/{id} message deleteMessage
//
// Deletes a message with an id.
//
//	---
//	produces: [application/json]
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: id
//	  in: path
//	  description: the message id
//	  required: true
//	  type: integer
//	  format: int64
//	responses:
//	  200:
//	    description: Ok
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
//	  404:
//	    description: Not Found
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) DeleteMessage(ctx *gin.Context) {
	withID(ctx, "id", func(id uint) {
		msg, err := a.DB.GetMessageByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if msg == nil {
			ctx.AbortWithError(404, errors.New("message does not exist"))
			return
		}
		app, err := a.DB.GetApplicationByID(msg.ApplicationID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && app.UserID == auth.GetUserID(ctx) {
			a.Scheduler.DeleteMessageSchedule(msg)
			successOrAbort(ctx, 500, a.DB.DeleteMessageByID(id))
		} else {
			ctx.AbortWithError(404, errors.New("message does not exist"))
		}
	})
}

// CreateMessage creates a message, authentication via application-token is required.
// swagger:operation POST /message message createMessage
//
// Create a message.
//
// __NOTE__: This API ONLY accepts an application token as authentication.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [appTokenAuthorizationHeader: [], appTokenHeader: [], appTokenQuery: []]
//	parameters:
//	- name: body
//	  in: body
//	  description: the message to add
//	  required: true
//	  schema:
//	    $ref: "#/definitions/Message"
//	responses:
//	  200:
//	    description: Ok
//	    schema:
//	      $ref: "#/definitions/Message"
//	  400:
//	    description: Bad Request
//	    schema:
//	        $ref: "#/definitions/Error"
//	  401:
//	    description: Unauthorized
//	    schema:
//	        $ref: "#/definitions/Error"
//	  403:
//	    description: Forbidden
//	    schema:
//	        $ref: "#/definitions/Error"
func (a *MessageAPI) CreateMessage(ctx *gin.Context) {
	message := model.MessageExternal{}
	if err := ctx.Bind(&message); err == nil {
		application, err := a.DB.GetApplicationByToken(auth.GetTokenID(ctx))
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		message.ApplicationID = application.ID
		if strings.TrimSpace(message.Title) == "" {
			message.Title = application.Name
		}

		if message.Priority == nil {
			message.Priority = &application.DefaultPriority
		}

		message.Date = timeNow()
		message.ID = 0
		msgInternal := toInternalMessage(&message)
		if success := successOrAbort(ctx, 500, a.DB.CreateMessage(msgInternal)); !success {
			return
		}
		if message.PostponedAt != nil {
			a.Scheduler.ScheduleMessage(msgInternal.ID, *message.PostponedAt)
		} else {
			a.Notifier.Notify(auth.GetUserID(ctx), scheduler.ToExternalMessage(msgInternal))
		}
		ctx.JSON(200, scheduler.ToExternalMessage(msgInternal))
	}
}

// / postponed message
func (a *MessageAPI) postponeMessage(ctx *gin.Context, postponedAt *time.Time) {
	withID(ctx, "id", func(id uint) {
		msg, err := a.DB.GetMessageByID(id)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if msg == nil {
			ctx.AbortWithError(404, errors.New("message does not exist"))
			return
		}
		app, err := a.DB.GetApplicationByID(msg.ApplicationID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if app != nil && app.UserID == auth.GetUserID(ctx) {
			a.Scheduler.DeleteMessageSchedule(msg)
			if postponedAt != nil {
				a.Scheduler.ScheduleMessage(msg.ID, *postponedAt)
			}
			successOrAbort(ctx, 500, a.DB.UpdateMessagePostponement(id, postponedAt))
		} else {
			ctx.AbortWithError(404, errors.New("message does not exist"))
		}
	})
}

func (a *MessageAPI) PostponeMessage(ctx *gin.Context) {
	at := ctx.Query("at")
	if at == "" {
		ctx.AbortWithError(400, errors.New("at parameter is required"))
		return
	}
	postponedAt, err := time.Parse(time.RFC3339, at)
	if err != nil {
		ctx.AbortWithError(400, errors.New("invalid time format. use RFC3339 format"))
		return
	}
	if postponedAt.Before(timeNow()) {
		ctx.AbortWithError(400, errors.New("postponed time must be in the future"))
		return
	}
	a.postponeMessage(ctx, &postponedAt)
}

func (a *MessageAPI) DeleteMessagePostponement(ctx *gin.Context) {
	a.postponeMessage(ctx, nil)
}

func toInternalMessage(msg *model.MessageExternal) *model.Message {
	res := &model.Message{
		ID:            msg.ID,
		ApplicationID: msg.ApplicationID,
		Message:       msg.Message,
		Title:         msg.Title,
		Date:          msg.Date,
		PostponedAt:   msg.PostponedAt,
	}
	if msg.Priority != nil {
		res.Priority = *msg.Priority
	}

	if msg.Extras != nil {
		res.Extras, _ = json.Marshal(msg.Extras)
	}
	return res
}

func toExternalMessages(msg []*model.Message) []*model.MessageExternal {
	res := make([]*model.MessageExternal, len(msg))
	for i := range msg {
		res[i] = scheduler.ToExternalMessage(msg[i])
	}
	return res
}
