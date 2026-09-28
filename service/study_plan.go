package service

import (
	"context"
	"log/slog"
	"time"

	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// StudyPlanService - O'quv rejasi va push eslatma vaqti (BACKEND_STRUCTURE.md 2-bo'lim).
type StudyPlanService struct {
	storage st.Storage
	push    *PushService
}

func NewStudyPlanService(storage *st.Storage, push *PushService) *StudyPlanService {
	return &StudyPlanService{storage: *storage, push: push}
}

func (s *StudyPlanService) Upsert(ctx context.Context, req *m.StudyPlanUpsertReq) (*m.Void, error) {
	return s.storage.StudyPlanS.Upsert(req)
}

func (s *StudyPlanService) GetByUserId(ctx context.Context, userId string) (*m.StudyPlanRes, error) {
	return s.storage.StudyPlanS.GetByUserId(userId)
}

// tashkentLocation - study_plans.remind_time har doim Asia/Tashkent bo'yicha
// talqin qilinadi (loyihada boshqa joyda ham shu zona ishlatiladi, masalan
// storage/postgres/order.go). tzdata topilmasa fixed +5:00'ga tushadi.
func tashkentLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		slog.Warn("tzdata missing? fallback to fixed zone", "err", err)
		return time.FixedZone("Asia/Tashkent", 5*60*60)
	}
	return loc
}

// RunReminderScheduler - har daqiqada study_plans jadvalidagi barcha faol
// rejalarni tekshiradi: (remind_time - remind_minutes_before) joriy daqiqaga
// (Asia/Tashkent) to'g'ri kelgan foydalanuvchilarga EventStudyReminder push
// yuboradi. app.Run ichida alohida goroutine sifatida ishga tushiriladi;
// ctx bekor qilinsa to'xtaydi.
func (s *StudyPlanService) RunReminderScheduler(ctx context.Context) {
	if s.push == nil {
		slog.Warn("study plan reminder scheduler: push service yo'q, o'chirilgan")
		return
	}

	loc := tashkentLocation()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickReminders(ctx, loc)
		}
	}
}

func (s *StudyPlanService) tickReminders(ctx context.Context, loc *time.Location) {
	plans, err := s.storage.StudyPlanS.GetAllEnabled()
	if err != nil {
		slog.Error("study plan reminder: GetAllEnabled failed", "err", err)
		return
	}

	nowMinutes := minutesSinceMidnight(time.Now().In(loc))

	for _, p := range plans {
		if p.UserId == nil || p.RemindTime == nil {
			continue
		}
		remindMinutes, ok := parseHHMMToMinutes(*p.RemindTime)
		if !ok {
			slog.Warn("study plan reminder: invalid remind_time", "user_id", *p.UserId, "remind_time", *p.RemindTime)
			continue
		}
		before := 0
		if p.RemindMinutesBefore != nil {
			before = *p.RemindMinutesBefore
		}
		if !reminderDueNow(remindMinutes, before, nowMinutes) {
			continue
		}

		userId := *p.UserId
		if err := s.push.SendPush(ctx, m.PushTarget{Type: m.TargetUser, UserID: userId}, m.PushEvent{Type: m.EventStudyReminder}); err != nil {
			slog.Warn("study plan reminder push failed", "user_id", userId, "err", err)
		}
	}
}

// reminderDueNow - remindMinutes (kun ichidagi daqiqa, 0-1439) dan
// beforeMinutes oldingi lahza joriy daqiqaga to'g'ri kelsa true. Manfiy
// natija (masalan 00:05 dan 15 daqiqa oldin) kun boshiga o'raladi (23:50).
func reminderDueNow(remindMinutes, beforeMinutes, nowMinutes int) bool {
	const dayMinutes = 24 * 60
	target := ((remindMinutes-beforeMinutes)%dayMinutes + dayMinutes) % dayMinutes
	return target == nowMinutes
}

func minutesSinceMidnight(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

// parseHHMMToMinutes - Postgres "time" ustuni ::text bilan "20:00:00" yoki
// "20:00" ko'rinishida kelishi mumkin, ikkalasini ham qo'llab-quvvatlaydi.
func parseHHMMToMinutes(s string) (int, bool) {
	if t, err := time.Parse("15:04:05", s); err == nil {
		return t.Hour()*60 + t.Minute(), true
	}
	if t, err := time.Parse("15:04", s); err == nil {
		return t.Hour()*60 + t.Minute(), true
	}
	return 0, false
}
