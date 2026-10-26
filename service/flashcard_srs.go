package service

import (
	"fmt"
	"math"
	m "tibsphereai/pkg/models"
	"time"
)

// SRS - fleshkartaning intervalli takrorlash holati (SM-2 o'xshash, TZ_MVP jadvaliga moslangan).
type SRS struct {
	Ease     float64
	Interval int // kun
	Reps     int // ketma-ket muvaffaqiyatli takrorlar
	Lapses   int // unutilgan marta
}

const (
	minEase = 1.3
	maxEase = 3.0
	// againDelay - "Qayta ko'rish" bosilganda karta shu vaqtdan keyin bugunoq qaytadi.
	againDelay = 10 * time.Minute
)

// Schedule - baholashdan keyingi holat va keyingi ko'rsatish vaqti.
//
//	again: 10 daqiqadan keyin; reps=0, lapses+1, ease-0.2
//	hard : ~1 kun (takrorda prev*1.2); ease-0.15
//	good : 3 -> 7 -> 21 kun, keyin prev*ease
//	easy : 7 -> 14 kun, keyin prev*ease*1.3; ease+0.15
func Schedule(c SRS, rating string, now time.Time) (SRS, time.Time, error) {
	if c.Ease <= 0 {
		c.Ease = 2.5
	}
	prev := c.Interval
	switch rating {
	case m.FlashcardRatingAgain:
		c.Reps = 0
		c.Lapses++
		c.Interval = 0
		c.Ease = math.Max(minEase, c.Ease-0.2)
		return c, now.Add(againDelay), nil
	case m.FlashcardRatingHard:
		c.Ease = math.Max(minEase, c.Ease-0.15)
		if c.Reps == 0 || prev <= 0 {
			c.Interval = 1
		} else {
			c.Interval = max(1, int(math.Round(float64(prev)*1.2)))
		}
		c.Reps++
	case m.FlashcardRatingGood:
		switch {
		case c.Reps == 0:
			c.Interval = 3
		case c.Reps == 1:
			c.Interval = 7
		case c.Reps == 2:
			c.Interval = 21
		default:
			c.Interval = max(prev+1, int(math.Round(float64(prev)*c.Ease)))
		}
		c.Reps++
	case m.FlashcardRatingEasy:
		switch {
		case c.Reps == 0:
			c.Interval = 7
		case c.Reps == 1:
			c.Interval = 14
		default:
			c.Interval = max(prev+1, int(math.Round(float64(prev)*c.Ease*1.3)))
		}
		c.Ease = math.Min(maxEase, c.Ease+0.15)
		c.Reps++
	default:
		return c, now, fmt.Errorf("invalid rating %q (again|hard|good|easy)", rating)
	}
	return c, now.Add(time.Duration(c.Interval) * 24 * time.Hour), nil
}

// IntervalLabel - tugmalar uchun keyingi interval (masalan "10 daq", "3 kun") frontend hisoblashi uchun
// raqamlar bilan birga beriladi; bu yerda faqat kun soni.
func PreviewIntervals(c SRS, now time.Time) map[string]int {
	out := map[string]int{}
	for _, r := range []string{m.FlashcardRatingAgain, m.FlashcardRatingHard, m.FlashcardRatingGood, m.FlashcardRatingEasy} {
		n, due, err := Schedule(c, r, now)
		if err != nil {
			continue
		}
		if r == m.FlashcardRatingAgain {
			out[r] = 0
			_ = due
		} else {
			out[r] = n.Interval
		}
	}
	return out
}
