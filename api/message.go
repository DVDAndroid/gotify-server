package api

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/rs/zerolog/log"
)

// The MessageDatabase interface for encapsulating database access.
type MessageDatabase interface {
	GetMessagesByApplicationSince(appID uint, limit int, since uint, postponed *string) ([]*model.Message, error)
	GetApplicationByID(id uint) (*model.Application, error)
	GetMessagesByUserSince(userID uint, limit int, since uint, postponed *string) ([]*model.Message, error)
	UpdateMessagePostponement(id uint, postponedAt *time.Time) error
	MarkMessagePostponementDelivered(id uint) error
	GetMessagesToDeliver(now time.Time) ([]*model.Message, error)
	DeleteMessageByID(id uint) error
	GetMessageByID(id uint) (*model.Message, error)
	DeleteMessagesByUser(userID uint) error
	DeleteMessagesByApplication(applicationID uint) error
	CreateMessage(message *model.Message) error
}

var timeNow = time.Now

// Notifier notifies when a new message was created.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}

// The MessageAPI provides handlers for managing messages.
type MessageAPI struct {
	DB       MessageDatabase
	Notifier Notifier
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
//	  description: "return all (all), only postponed (true) or only not postponed (false) messages"
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
		query := url.Values{}
		query.Add("limit", strconv.Itoa(paging.Limit))
		query.Add("since", strconv.FormatUint(uint64(since), 10))
		next = ctx.Request.URL.Path + "?" + query.Encode()
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
//	  description: "return all (all), only postponed (true) or only not postponed (false) messages"
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
			successOrAbort(ctx, 500, a.DB.DeleteMessageByID(id))
		} else {
			ctx.AbortWithError(404, errors.New("message does not exist"))
		}
	})
}

// CreateMessage creates a message, authentication via application token, client token, or basic auth is required.
// swagger:operation POST /message message createMessage
//
// Create a message.
//
// __NOTE__: When authenticating with a client token or basic auth, the request body
// must include "appid" referencing an application owned by the authenticated user.
// When authenticating with an application token, the application is derived from the
// token and any "appid" in the body is ignored.
//
//	---
//	consumes: [application/json]
//	produces: [application/json]
//	security: [appTokenAuthorizationHeader: [], appTokenHeader: [], appTokenQuery: [], clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: body
//	  in: body
//	  description: the message to add
//	  required: true
//	  schema:
//	    $ref: "#/definitions/CreateMessage"
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
	message := model.CreateMessage{}
	if err := ctx.Bind(&message); err != nil {
		return
	}

	app := auth.GetApplication(ctx)
	if app == nil {
		if message.ApplicationID == 0 {
			ctx.AbortWithError(400, errors.New("appid is required when not authenticating with an application token"))
			return
		}
		fetchedApp, err := a.DB.GetApplicationByID(message.ApplicationID)
		if success := successOrAbort(ctx, 500, err); !success {
			return
		}
		if fetchedApp == nil || fetchedApp.UserID != auth.GetUserID(ctx) {
			ctx.AbortWithError(400, errors.New("appid not found"))
			return
		}
		app = fetchedApp
	}

	message.ApplicationID = app.ID
	if strings.TrimSpace(message.Title) == "" {
		message.Title = app.Name
	}

	if message.Priority == nil {
		message.Priority = &app.DefaultPriority
	}

	msgInternal := toInternalMessage(&message)
	if success := successOrAbort(ctx, 500, a.DB.CreateMessage(msgInternal)); !success {
		return
	}
	if msgInternal.PostponedAt != nil {
		a.schedule(msgInternal.ID, *msgInternal.PostponedAt)
	} else {
		a.Notifier.Notify(auth.GetUserID(ctx), toExternalMessage(msgInternal))
	}
	ctx.JSON(200, toExternalMessage(msgInternal))
}

// PostponeMessage postpones the push of a message to the given date.
// swagger:operation POST /message/{id}/postpone message postponeMessage
//
// Postpone a message.
//
//	---
//	security: [clientTokenAuthorizationHeader: [], clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
//	parameters:
//	- name: id
//	  in: path
//	  description: the message id
//	  required: true
//	  type: integer
//	  format: int64
//	- name: at
//	  in: query
//	  description: "the date to postpone the message to, in RFC3339 format with seconds, e.g. 2026-11-13T10:00:00+01:00. Must be in the future."
//	  required: true
//	  type: string
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
func (a *MessageAPI) PostponeMessage(ctx *gin.Context) {
	at := ctx.Query("at")
	if at == "" {
		ctx.AbortWithError(400, errors.New("at parameter is required"))
		return
	}
	postponedAt, err := time.Parse(time.RFC3339, at)
	if err != nil {
		ctx.AbortWithError(400, errors.New("invalid time format, use RFC3339"))
		return
	}
	if postponedAt.Before(timeNow()) {
		ctx.AbortWithError(400, errors.New("postponed time must be in the future"))
		return
	}
	a.postponeMessage(ctx, &postponedAt)
}

// DeleteMessagePostponement removes the postponement of a message.
// swagger:operation DELETE /message/{id}/postpone message deleteMessagePostponement
//
// Remove the postponement of a message.
//
//	---
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
func (a *MessageAPI) DeleteMessagePostponement(ctx *gin.Context) {
	a.postponeMessage(ctx, nil)
}

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
		if app == nil || app.UserID != auth.GetUserID(ctx) {
			ctx.AbortWithError(404, errors.New("message does not exist"))
			return
		}
		if success := successOrAbort(ctx, 500, a.DB.UpdateMessagePostponement(id, postponedAt)); !success {
			return
		}
		if postponedAt != nil {
			a.schedule(id, *postponedAt)
		}
	})
}

// schedule pushes the message to its user at the given date. Timers are never cancelled:
// deliver ignores a timer if the message was deleted or postponed to another date meanwhile.
func (a *MessageAPI) schedule(id uint, at time.Time) {
	time.AfterFunc(time.Until(at), func() { a.deliver(id, at) })
}

func (a *MessageAPI) deliver(id uint, at time.Time) {
	msg, err := a.DB.GetMessageByID(id)
	if err != nil {
		log.Error().Err(err).Uint("message", id).Msg("Cannot load postponed message")
		return
	}
	// ponytail: 1s tolerance, because some databases do not store sub-second precision.
	if msg == nil || msg.PostponedAt == nil || msg.PostponedAt.Sub(at).Abs() >= time.Second {
		return
	}
	app, err := a.DB.GetApplicationByID(msg.ApplicationID)
	if err != nil || app == nil {
		log.Error().Err(err).Uint("message", id).Msg("Cannot load application of postponed message")
		return
	}
	a.Notifier.Notify(app.UserID, toExternalMessage(msg))
	if err := a.DB.MarkMessagePostponementDelivered(id); err != nil {
		log.Error().Err(err).Uint("message", id).Msg("Cannot mark postponed message as delivered")
	}
}

// ScheduleAll schedules all postponed messages which were not delivered yet, call it on startup.
// Messages missed while the server was not running are delivered immediately.
func (a *MessageAPI) ScheduleAll() error {
	messages, err := a.DB.GetMessagesToDeliver(timeNow())
	for _, msg := range messages {
		a.schedule(msg.ID, *msg.PostponedAt)
	}
	return err
}

func toInternalMessage(msg *model.CreateMessage) *model.Message {
	res := &model.Message{
		ApplicationID:   msg.ApplicationID,
		Message:         msg.Message,
		Title:           msg.Title,
		Date:            timeNow(),
		PostponedAt:     msg.PostponedAt,
		PostponePending: msg.PostponedAt != nil,
	}
	if msg.Priority != nil {
		res.Priority = *msg.Priority
	}

	if msg.Extras != nil {
		res.Extras, _ = json.Marshal(msg.Extras)
	}
	return res
}

func toExternalMessage(msg *model.Message) *model.MessageExternal {
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
		res.Extras = make(map[string]any)
		json.Unmarshal(msg.Extras, &res.Extras)
	}
	return res
}

func toExternalMessages(msg []*model.Message) []*model.MessageExternal {
	res := make([]*model.MessageExternal, len(msg))
	for i := range msg {
		res[i] = toExternalMessage(msg[i])
	}
	return res
}
