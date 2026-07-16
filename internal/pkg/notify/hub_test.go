package notify

import (
	"testing"

	"acquisition_backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubPublishReachesSubscriber(t *testing.T) {
	h := NewHub()
	userID := uuid.New()

	ch, unsub := h.Subscribe(userID)
	require.NotNil(t, ch)
	defer unsub()

	n := &domain.Notification{ID: uuid.New(), RecipientID: userID, Title: "test"}
	h.Publish(n)

	got := <-ch
	assert.Equal(t, n.ID, got.ID)
}

func TestHubPublishSkipsOtherUsers(t *testing.T) {
	h := NewHub()
	ch, unsub := h.Subscribe(uuid.New())
	require.NotNil(t, ch)
	defer unsub()

	h.Publish(&domain.Notification{ID: uuid.New(), RecipientID: uuid.New()})
	assert.Empty(t, ch)
}

func TestHubUnsubscribeClosesChannel(t *testing.T) {
	h := NewHub()
	userID := uuid.New()
	ch, unsub := h.Subscribe(userID)
	require.NotNil(t, ch)

	unsub()
	_, open := <-ch
	assert.False(t, open)

	// Unsubscribe-ийн дараах publish паник үүсгэхгүй байх ёстой.
	h.Publish(&domain.Notification{RecipientID: userID})
	// Давхар unsubscribe мөн аюулгүй.
	unsub()
}

func TestHubPerUserConnectionLimit(t *testing.T) {
	h := NewHub()
	userID := uuid.New()

	var cleanups []func()
	for i := 0; i < maxSubsPerUser; i++ {
		ch, unsub := h.Subscribe(userID)
		require.NotNil(t, ch, "connection %d within limit must succeed", i+1)
		cleanups = append(cleanups, unsub)
	}
	over, unsub := h.Subscribe(userID)
	assert.Nil(t, over, "connection over the limit must be rejected")
	unsub()

	for _, c := range cleanups {
		c()
	}
	// Хязгаар суллагдсаны дараа дахин холбогдож болно.
	ch, unsub := h.Subscribe(userID)
	assert.NotNil(t, ch)
	unsub()
}

func TestHubFullBufferDoesNotBlock(t *testing.T) {
	h := NewHub()
	userID := uuid.New()
	ch, unsub := h.Subscribe(userID)
	require.NotNil(t, ch)
	defer unsub()

	// Буфер дүүрсэн ч Publish блоклохгүй, илүүдлийг алгасна.
	for i := 0; i < chanBuffer+10; i++ {
		h.Publish(&domain.Notification{ID: uuid.New(), RecipientID: userID})
	}
	assert.Len(t, ch, chanBuffer)
}
