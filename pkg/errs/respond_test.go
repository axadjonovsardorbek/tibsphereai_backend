package errs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// TestIsInvalidInputSyntax - Postgres "invalid input syntax for type uuid"
// (SQLSTATE 22P02, masalan /mobile/simulation/sim-1/finish kabi UUID
// bo'lmagan path param yuborilganda) to'g'ri aniqlanishi kerak - shu orqali
// Respond() xom SQL xatosi o'rniga tushunarli ErrInvalidId qaytaradi.
//
// Diqqat: bu yerda to'g'ridan-to'g'ri Bad()/Respond() chaqirilmaydi - ular
// muvaffaqiyatsiz so'rovda notify()'ni ishga tushiradi, u esa real Telegram
// guruhga xabar yuboradi (test uchun yaramaydi, xom tarmoq chaqiruvi bo'ladi).
func TestIsInvalidInputSyntax(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "uuid syntax error",
			err:  &pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type uuid: "sim-1"`},
			want: true,
		},
		{
			name: "wrapped uuid syntax error (errors.As unwraps through fmt.Errorf)",
			err:  fmt.Errorf("storage: %w", &pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type uuid: "sim-1"`}),
			want: true,
		},
		{
			name: "different pg error code",
			err:  &pgconn.PgError{Code: "23505", Message: "duplicate key value"},
			want: false,
		},
		{
			name: "plain non-pg error",
			err:  errors.New("case not found"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isInvalidInputSyntax(tt.err))
		})
	}
}
