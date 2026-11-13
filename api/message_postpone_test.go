package api

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chanNotifier is safe to use with timers, which notify from another goroutine.
type chanNotifier struct {
	users chan uint
	msgs  chan *model.MessageExternal
}

func newChanNotifier() *chanNotifier {
	return &chanNotifier{users: make(chan uint, 10), msgs: make(chan *model.MessageExternal, 10)}
}

func (n *chanNotifier) Notify(userID uint, msg *model.MessageExternal) {
	n.users <- userID
	n.msgs <- msg
}

func (s *MessageSuite) postponeRequest(method string, at string) {
	target := "/message/50/postpone"
	if at != "" {
		target += "?at=" + url.QueryEscape(at)
	}
	s.ctx.Request = httptest.NewRequest(method, target, nil)
	s.ctx.Params = gin.Params{{Key: "id", Value: "50"}}
}

func (s *MessageSuite) Test_PostponeMessage() {
	s.db.User(6).App(1).Message(50)
	at := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	test.WithUser(s.ctx, 6)
	s.postponeRequest("POST", at.Format(time.RFC3339))
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 200, s.recorder.Code)
	msg, err := s.db.GetMessageByID(50)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), msg.PostponedAt)
	assert.True(s.T(), at.Equal(*msg.PostponedAt), "expected %s, got %s", at, msg.PostponedAt)
}

func (s *MessageSuite) Test_PostponeMessage_missingAt() {
	s.db.User(6).App(1).Message(50)

	test.WithUser(s.ctx, 6)
	s.postponeRequest("POST", "")
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 400, s.recorder.Code)
}

func (s *MessageSuite) Test_PostponeMessage_invalidFormat() {
	s.db.User(6).App(1).Message(50)

	test.WithUser(s.ctx, 6)
	s.postponeRequest("POST", "tomorrow")
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 400, s.recorder.Code)
}

func (s *MessageSuite) Test_PostponeMessage_inThePast() {
	s.db.User(6).App(1).Message(50)

	test.WithUser(s.ctx, 6)
	s.postponeRequest("POST", time.Now().Add(-time.Hour).Format(time.RFC3339))
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 400, s.recorder.Code)
	msg, _ := s.db.GetMessageByID(50)
	assert.Nil(s.T(), msg.PostponedAt)
}

func (s *MessageSuite) Test_PostponeMessage_notExisting() {
	s.db.User(6)

	test.WithUser(s.ctx, 6)
	s.postponeRequest("POST", time.Now().Add(time.Hour).Format(time.RFC3339))
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 404, s.recorder.Code)
}

func (s *MessageSuite) Test_PostponeMessage_notOwner() {
	s.db.User(1).App(10).Message(50)
	s.db.User(2)

	test.WithUser(s.ctx, 2)
	s.postponeRequest("POST", time.Now().Add(time.Hour).Format(time.RFC3339))
	s.a.PostponeMessage(s.ctx)

	assert.Equal(s.T(), 404, s.recorder.Code)
	msg, _ := s.db.GetMessageByID(50)
	assert.Nil(s.T(), msg.PostponedAt)
}

func (s *MessageSuite) Test_DeleteMessagePostponement() {
	s.db.User(6).App(1).Message(50)
	at := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, &at))

	test.WithUser(s.ctx, 6)
	s.postponeRequest("DELETE", "")
	s.a.DeleteMessagePostponement(s.ctx)

	assert.Equal(s.T(), 200, s.recorder.Code)
	msg, _ := s.db.GetMessageByID(50)
	assert.Nil(s.T(), msg.PostponedAt)
}

func (s *MessageSuite) Test_CreateMessage_postponedIsNotNotifiedImmediately() {
	at := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	auth.RegisterApplication(s.ctx, s.db.User(4).NewAppWithToken(7, "app-token"))
	body := `{"message": "mymessage", "postponed_at": "` + at.Format(time.RFC3339) + `"}`
	s.ctx.Request = httptest.NewRequest("POST", "/message", strings.NewReader(body))
	s.ctx.Request.Header.Set("Content-Type", "application/json")

	s.a.CreateMessage(s.ctx)

	assert.Equal(s.T(), 200, s.recorder.Code)
	assert.Nil(s.T(), s.notifiedMessage)
	msgs, err := s.db.GetMessagesByApplication(7)
	require.NoError(s.T(), err)
	require.Len(s.T(), msgs, 1)
	require.NotNil(s.T(), msgs[0].PostponedAt)
	assert.True(s.T(), at.Equal(*msgs[0].PostponedAt))
}

func (s *MessageSuite) Test_GetMessages_filterPostponed() {
	user := s.db.User(5)
	user.App(1).Message(1)
	user.App(1).Message(2)
	at := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(2, &at))

	ids := func(query string) []uint {
		s.recorder = httptest.NewRecorder()
		s.ctx, _ = gin.CreateTestContext(s.recorder)
		s.ctx.Request = httptest.NewRequest("GET", "/irrelevant", nil)
		s.withURL("http", "example.com", "/messages", query)
		test.WithUser(s.ctx, 5)
		s.a.GetMessages(s.ctx)
		var res model.PagedMessages
		require.NoError(s.T(), json.Unmarshal(s.recorder.Body.Bytes(), &res))
		var out []uint
		for _, m := range res.Messages {
			out = append(out, m.ID)
		}
		return out
	}

	assert.Equal(s.T(), []uint{2, 1}, ids(""))
	assert.Equal(s.T(), []uint{2, 1}, ids("postponed=all"))
	assert.Equal(s.T(), []uint{2}, ids("postponed=true"))
	assert.Equal(s.T(), []uint{1}, ids("postponed=false"))
}

// deliver must notify the owner of the application, not the user whose id equals the application id.
func (s *MessageSuite) Test_deliver_notifiesApplicationOwner() {
	s.db.User(7).App(3).Message(50)
	at := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, &at))

	s.a.deliver(50, at)

	require.NotNil(s.T(), s.notifiedMessage)
	assert.Equal(s.T(), uint(7), s.notifiedUser)
	assert.Equal(s.T(), uint(50), s.notifiedMessage.ID)

	msg, err := s.db.GetMessageByID(50)
	require.NoError(s.T(), err)
	assert.False(s.T(), msg.PostponePending, "delivered messages must not be delivered again after a restart")
	assert.NotNil(s.T(), msg.PostponedAt)
}

func (s *MessageSuite) Test_deliver_ignoresStaleTimers() {
	s.db.User(7).App(3).Message(50)
	at := time.Now().Add(time.Hour)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, &at))

	// postponed to another date meanwhile
	s.a.deliver(50, at.Add(time.Hour))
	assert.Nil(s.T(), s.notifiedMessage)

	// postponement removed meanwhile
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, nil))
	s.a.deliver(50, at)
	assert.Nil(s.T(), s.notifiedMessage)

	// deleted meanwhile
	require.NoError(s.T(), s.db.DeleteMessageByID(50))
	s.a.deliver(50, at)
	assert.Nil(s.T(), s.notifiedMessage)
}

func (s *MessageSuite) Test_schedule_deliversAtTheGivenDate() {
	s.db.User(7).App(3).Message(50)
	at := time.Now().Add(100 * time.Millisecond)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, &at))
	notifier := newChanNotifier()
	api := &MessageAPI{DB: s.db, Notifier: notifier}

	api.schedule(50, at)

	select {
	case user := <-notifier.users:
		assert.Equal(s.T(), uint(7), user)
		assert.Equal(s.T(), uint(50), (<-notifier.msgs).ID)
	case <-time.After(3 * time.Second):
		s.T().Fatal("postponed message was not delivered")
	}
}

// After a restart the missed messages are delivered, the already delivered ones are not.
func (s *MessageSuite) Test_ScheduleAll_deliversMissedAndFutureMessages() {
	app := s.db.User(7).App(3)
	app.Message(50)
	app.Message(51)
	app.Message(52)
	past := time.Now().Add(-time.Hour)
	soon := time.Now().Add(100 * time.Millisecond)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(50, &past)) // missed while the server was down
	require.NoError(s.T(), s.db.UpdateMessagePostponement(51, &soon)) // still in the future
	require.NoError(s.T(), s.db.UpdateMessagePostponement(52, &past))
	require.NoError(s.T(), s.db.MarkMessagePostponementDelivered(52)) // delivered before the restart
	notifier := newChanNotifier()
	api := &MessageAPI{DB: s.db, Notifier: notifier}

	require.NoError(s.T(), api.ScheduleAll())

	var delivered []uint
	for len(delivered) < 2 {
		select {
		case <-notifier.users:
			delivered = append(delivered, (<-notifier.msgs).ID)
		case <-time.After(3 * time.Second):
			s.T().Fatalf("postponed messages were not delivered, got only %v", delivered)
		}
	}
	assert.ElementsMatch(s.T(), []uint{50, 51}, delivered)
	select {
	case <-notifier.users:
		s.T().Fatal("already delivered message must not be delivered again")
	case <-time.After(300 * time.Millisecond):
	}
}
