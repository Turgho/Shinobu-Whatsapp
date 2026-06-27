package history

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(":memory:", zap.NewNop())
	require.NoError(t, err)
	return store
}

func TestNewStore_InMemory(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	assert.NotNil(t, store.db)
	assert.NotNil(t, store.UserMemory)
}

func TestSaveAndRecentMessages(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "chat1@g.us", "Usuário", "Olá"))
	require.NoError(t, store.Save(ctx, "chat1@g.us", "Shinobu", "Olá, como posso ajudar?"))
	require.NoError(t, store.Save(ctx, "chat2@g.us", "Usuário", "Outro chat"))

	msgs, err := store.RecentMessages(ctx, "chat1@g.us", 10, 5*time.Minute)
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	// Ordem cronológica: primeiro "user", depois "assistant"
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "Olá", msgs[0].Content)
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "Olá, como posso ajudar?", msgs[1].Content)
}

func TestRecentMessages_RespectsLimit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	for range 20 {
		require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "msg"))
	}

	msgs, err := store.RecentMessages(ctx, "chat@g.us", 5, 5*time.Minute)
	require.NoError(t, err)
	assert.Len(t, msgs, 5)
}

func TestSaveAndGetSummary(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	summary, err := store.GetSummary(ctx, "chat@g.us")
	require.NoError(t, err)
	assert.Equal(t, "", summary)

	require.NoError(t, store.SetSummary(ctx, "chat@g.us", "Usuário perguntou sobre o tempo"))

	summary, err = store.GetSummary(ctx, "chat@g.us")
	require.NoError(t, err)
	assert.Equal(t, "Usuário perguntou sobre o tempo", summary)
}

func TestNeedsSummary(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	needs, err := store.NeedsSummary(ctx, "chat@g.us", 5, 5*time.Minute)
	require.NoError(t, err)
	assert.False(t, needs)

	for range 5 {
		require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "msg"))
	}

	needs, err = store.NeedsSummary(ctx, "chat@g.us", 5, 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, needs)
}

func TestTranscriptRecent_MergeConsecutive(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "oi"))
	require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "tudo bem?"))
	require.NoError(t, store.Save(ctx, "chat@g.us", "Shinobu", "tudo e vc?"))
	require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "tudo"))

	transcript, err := store.TranscriptRecent(ctx, "chat@g.us", 10, 5*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, transcript, "oi | tudo bem?")
	assert.Contains(t, transcript, "tudo e vc?")
	assert.Contains(t, transcript, "tudo")
}

func TestCountRecentMessages(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()
	ctx := context.Background()

	for range 3 {
		require.NoError(t, store.Save(ctx, "chat@g.us", "Usuário", "msg"))
	}

	count, err := store.CountRecentMessages(ctx, "chat@g.us", 5*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestFormatRelativeConversationTime(t *testing.T) {
	now := time.Now()

	assert.Equal(t, "?", formatRelativeConversationTime(time.Time{}))
	assert.Equal(t, "agora", formatRelativeConversationTime(now))
	assert.Equal(t, "há 5m", formatRelativeConversationTime(now.Add(-5*time.Minute)))
	assert.Equal(t, "há 2h", formatRelativeConversationTime(now.Add(-2*time.Hour)))

	old := now.Add(-72 * time.Hour)
	result := formatRelativeConversationTime(old)
	assert.Contains(t, result, "/")
	assert.Contains(t, result, ":")
}

func TestParseSQLiteSentAt(t *testing.T) {
	assert.True(t, parseSQLiteSentAt("").IsZero())
	assert.True(t, parseSQLiteSentAt("   ").IsZero())

	expected := time.Date(2025, 6, 15, 10, 30, 0, 0, time.Local)
	result := parseSQLiteSentAt("2025-06-15 10:30:00")
	assert.Equal(t, expected, result)

	assert.True(t, parseSQLiteSentAt("invalido").IsZero())
}
