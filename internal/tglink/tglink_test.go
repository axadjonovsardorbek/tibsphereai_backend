package tglink

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rdbT(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()}), mr
}

func TestToken_Unique_Hex(t *testing.T) {
	a, b := NewToken(), NewToken()
	assert.NotEqual(t, a, b)
	assert.Len(t, a, 48)
	assert.True(t, isHex(a))
}

func TestParseStartPayload(t *testing.T) {
	tok := NewToken()
	p, got, ok := ParseStartPayload("login_" + tok)
	assert.True(t, ok)
	assert.Equal(t, PurposeLogin, p)
	assert.Equal(t, tok, got)
	p, _, ok = ParseStartPayload("phone_" + tok)
	assert.True(t, ok)
	assert.Equal(t, PurposePhone, p)

	for _, bad := range []string{"", "login", "login_", "login_short", "login_ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", "other_" + tok, "login_" + tok + "x$"} {
		_, _, ok := ParseStartPayload(bad)
		assert.False(t, ok, bad)
	}
}

func TestFlow_Login_ConfirmConsume(t *testing.T) {
	rdb, _ := rdbT(t)
	tok := NewToken()
	// yaratilmagan token tasdiqlanmaydi (bot o'zi token o'ylab topa olmaydi)
	assert.ErrorIs(t, Confirm(rdb, PurposeLogin, tok, Info{Phone: "+998901"}), ErrNotFound)

	require.NoError(t, Create(rdb, PurposeLogin, tok, "", false))
	in, ok, err := Consume(rdb, PurposeLogin, tok)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, StatusPending, in.Status)

	require.NoError(t, Confirm(rdb, PurposeLogin, tok, Info{Phone: "+998901234567", Name: "Ali", TgID: 42}))
	assert.ErrorIs(t, Confirm(rdb, PurposeLogin, tok, Info{Phone: "+998999"}), ErrNotPending, "ikkinchi marta tasdiqlab bo'lmaydi")

	in, ok, err = Consume(rdb, PurposeLogin, tok)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "+998901234567", in.Phone)
	assert.Equal(t, int64(42), in.TgID)
	// bir martalik
	_, _, err = Consume(rdb, PurposeLogin, tok)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFlow_Phone_OwnerAndExpiry(t *testing.T) {
	rdb, mr := rdbT(t)
	tok := NewToken()
	require.NoError(t, Create(rdb, PurposePhone, tok, "user-1", false))
	require.NoError(t, Confirm(rdb, PurposePhone, tok, Info{Phone: "+998901234567"}))
	in, err := Get(rdb, PurposePhone, tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", in.Owner, "egasi bot yozganda ham saqlanadi")
	// login va phone tokenlari alohida nom fazosida
	_, err = Get(rdb, PurposeLogin, tok)
	assert.ErrorIs(t, err, ErrNotFound)
	mr.FastForward(TTL + time.Second)
	_, err = Get(rdb, PurposePhone, tok)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestKnownUser_And_Waiting(t *testing.T) {
	rdb, _ := rdbT(t)
	_, ok := KnownUser(rdb, 7)
	assert.False(t, ok)
	RememberUser(rdb, 7, Known{Phone: "+998901", Name: "Ali"})
	k, ok := KnownUser(rdb, 7)
	assert.True(t, ok)
	assert.Equal(t, "Ali", k.Name)

	tok := NewToken()
	SetWaiting(rdb, 100, PurposeLogin, tok)
	p, got, ok := PopWaiting(rdb, 100)
	assert.True(t, ok)
	assert.Equal(t, PurposeLogin, p)
	assert.Equal(t, tok, got)
	_, _, ok = PopWaiting(rdb, 100)
	assert.False(t, ok, "bir marta olinadi")
}
