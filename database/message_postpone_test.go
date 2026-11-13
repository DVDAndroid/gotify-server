package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *DatabaseSuite) TestMessagePostponement() {
	user := &model.User{Name: "test", Pass: []byte{1}}
	require.NoError(s.T(), s.db.CreateUser(user))
	app := &model.Application{UserID: user.ID, Token: "A0000000000", Name: "app"}
	require.NoError(s.T(), s.db.CreateApplication(app))

	now := time.Now()
	normal := &model.Message{ApplicationID: app.ID, Message: "normal", Date: now}
	soon := &model.Message{ApplicationID: app.ID, Message: "soon", Date: now}
	past := &model.Message{ApplicationID: app.ID, Message: "past", Date: now}
	for _, m := range []*model.Message{normal, soon, past} {
		require.NoError(s.T(), s.db.CreateMessage(m))
		assert.Nil(s.T(), m.PostponedAt)
	}
	soonAt, pastAt := now.Add(time.Hour), now.Add(-time.Hour)
	require.NoError(s.T(), s.db.UpdateMessagePostponement(soon.ID, &soonAt))
	require.NoError(s.T(), s.db.UpdateMessagePostponement(past.ID, &pastAt))

	stored, err := s.db.GetMessageByID(soon.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), stored.PostponedAt)
	assert.WithinDuration(s.T(), soonAt, *stored.PostponedAt, time.Second)

	ids := func(msgs []*model.Message, err error) []uint {
		require.NoError(s.T(), err)
		var out []uint
		for _, m := range msgs {
			out = append(out, m.ID)
		}
		return out
	}
	str := func(v string) *string { return &v }

	all := []uint{past.ID, soon.ID, normal.ID}
	onlyPostponed := []uint{past.ID, soon.ID}
	onlyNormal := []uint{normal.ID}
	for filter, expected := range map[*string][]uint{
		nil:             all,
		str("all"):      all,
		str("true"):     onlyPostponed,
		str(""):         onlyPostponed,
		str("false"):    onlyNormal,
		str("whatever"): onlyNormal,
	} {
		assert.Equal(s.T(), expected, ids(s.db.GetMessagesByUserSince(user.ID, 50, 0, filter)), "user filter")
		assert.Equal(s.T(), expected, ids(s.db.GetMessagesByApplicationSince(app.ID, 50, 0, filter)), "application filter")
	}

	// postponing marks the message as pending: future and missed (pending, in the past) messages are found
	assert.True(s.T(), stored.PostponePending)
	assert.ElementsMatch(s.T(), []uint{past.ID, soon.ID}, ids(s.db.GetMessagesToDeliver(now)))

	// delivered messages are not found anymore, unless they are postponed to the future
	require.NoError(s.T(), s.db.MarkMessagePostponementDelivered(past.ID))
	assert.Equal(s.T(), []uint{soon.ID}, ids(s.db.GetMessagesToDeliver(now)))
	require.NoError(s.T(), s.db.MarkMessagePostponementDelivered(soon.ID))
	assert.Equal(s.T(), []uint{soon.ID}, ids(s.db.GetMessagesToDeliver(now)), "legacy future messages have no pending flag")
	stored, err = s.db.GetMessageByID(past.ID)
	require.NoError(s.T(), err)
	assert.False(s.T(), stored.PostponePending)
	assert.NotNil(s.T(), stored.PostponedAt, "the postponed date is kept after the delivery")

	// postponement can be removed
	require.NoError(s.T(), s.db.UpdateMessagePostponement(soon.ID, nil))
	stored, err = s.db.GetMessageByID(soon.ID)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), stored.PostponedAt)
	assert.False(s.T(), stored.PostponePending)
	assert.Empty(s.T(), ids(s.db.GetMessagesToDeliver(now)))
}
