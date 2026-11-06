package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// touchStreak - kunlik faollik (streak): bugun birinchi marta faol bo'lsa va kecha ham faol
// bo'lgan bo'lsa +1, aks holda 1 dan boshlanadi. Bir kunda takroriy chaqiruv hech narsa o'zgartirmaydi.
// Kun chegarasi - Asia/Tashkent. Xato asosiy oqimni to'xtatmaydi.
func touchStreak(ctx context.Context, db *pgxpool.Pool, userId string) {
	if db == nil || userId == "" {
		return
	}
	touchActive(ctx, db, userId)
	_, err := db.Exec(ctx, `
		UPDATE users SET
			streak_count = CASE
				WHEN streak_last_day = (now() AT TIME ZONE 'Asia/Tashkent')::date - 1 THEN streak_count + 1
				ELSE 1 END,
			longest_streak = GREATEST(longest_streak, CASE
				WHEN streak_last_day = (now() AT TIME ZONE 'Asia/Tashkent')::date - 1 THEN streak_count + 1
				ELSE 1 END),
			streak_last_day = (now() AT TIME ZONE 'Asia/Tashkent')::date
		WHERE id = $1 AND deleted_at = 0
		  AND (streak_last_day IS NULL OR streak_last_day < (now() AT TIME ZONE 'Asia/Tashkent')::date)
	`, userId)
	if err != nil {
		fmt.Println("failed to touch streak:", err)
	}
}

// touchActive - users.last_active_at: haqiqiy faollik (sessiya boshlash, javob, fleshkarta
// takrorlash) vaqti - login yoki profil ochish emas. Admin statistikasi shundan o'qiydi.
func touchActive(ctx context.Context, db *pgxpool.Pool, userId string) {
	if db == nil || userId == "" {
		return
	}
	if _, err := db.Exec(ctx, `UPDATE users SET last_active_at = now() WHERE id = $1`, userId); err != nil {
		fmt.Println("failed to touch last_active_at:", err)
	}
}
