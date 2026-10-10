package handler

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"tibsphereai/internal/ws"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// StartSimulation godoc
// @Summary Sessiyani boshlash (BACKEND_STRUCTURE.md 4-bo'lim)
// @Tags simulation
// @Accept json
// @Produce json
// @Param body body object{case_id=string} true "Case ID"
// @Success 200 {object} models.SimulationStartRes
// @Security BearerAuth
// @Router /mobile/simulation/start [post]
func (h *Handler) StartSimulation(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	var body struct {
		CaseId          string `json:"case_id"`
		DurationMinutes int    `json:"duration_minutes"`
		UseCoins        bool   `json:"use_coins"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Simulation.Start(context.Background(), &m.SimulationStartReq{UserId: userId, CaseId: body.CaseId, DurationMinutes: body.DurationMinutes, UseCoins: body.UseCoins})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// AddSimulationEvent godoc
// @Summary Anamnez/ko'rik/tahlil/dori qadamini yuborish (BACKEND_STRUCTURE.md 4-bo'lim)
// @Tags simulation
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param event body models.SimulationEventReq true "Event"
// @Success 200 {object} models.SimulationEventRes
// @Security BearerAuth
// @Router /mobile/simulation/{id}/event [post]
func (h *Handler) AddSimulationEvent(c *gin.Context) {
	var req m.SimulationEventReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.SessionId = c.Param("id")

	res, err := h.Clients.Simulation.AddEvent(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// FinishSimulation godoc
// @Summary Sessiyani yakunlash (BACKEND_STRUCTURE.md 4,5-bo'lim)
// @Tags simulation
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param body body object{reason=string} true "manual|health_zero|timeout"
// @Success 200 {object} models.SimulationFinishRes
// @Security BearerAuth
// @Router /mobile/simulation/{id}/finish [put]
func (h *Handler) FinishSimulation(c *gin.Context) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Simulation.Finish(context.Background(), &m.SimulationFinishReq{SessionId: c.Param("id"), Reason: body.Reason})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetByIdSimulation godoc
// @Tags simulation
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.SimulationSessionRes
// @Security BearerAuth
// @Router /mobile/simulation/{id} [get]
func (h *Handler) GetByIdSimulation(c *gin.Context) {
	res, err := h.Clients.Simulation.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetOngoingSimulations godoc
// @Summary Profile -> Ongoing (BACKEND_STRUCTURE.md 1-bo'lim)
// @Tags simulation
// @Produce json
// @Success 200 {object} models.SimulationSessionList
// @Security BearerAuth
// @Router /mobile/simulation/ongoing [get]
func (h *Handler) GetOngoingSimulations(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	data, _, err := h.Clients.Simulation.GetOngoing(context.Background(), &m.ById{Id: userId})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GetCompletedSimulations godoc
// @Summary Profile -> Completed (BACKEND_STRUCTURE.md 1-bo'lim)
// @Tags simulation
// @Produce json
// @Success 200 {object} models.SimulationSessionList
// @Security BearerAuth
// @Router /mobile/simulation/completed [get]
func (h *Handler) GetCompletedSimulations(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	data, _, err := h.Clients.Simulation.GetCompleted(context.Background(), &m.ById{Id: userId})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// SimulationWS godoc
// @Summary Real-vaqt vitallar monitori (BACKEND_STRUCTURE.md 4-bo'lim, WebSocket)
// @Tags simulation
// @Param id path string true "Session ID"
// @Router /mobile/simulation/{id}/ws [get]
func (h *Handler) SimulationWS(c *gin.Context) {
	sessionId := c.Param("id")

	conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	hub := ws.GetOrCreateHub(sessionId, func(sid string) (map[string]any, error) {
		session, err := h.Clients.Simulation.GetById(context.Background(), &m.ById{Id: sid})
		if err != nil || session == nil {
			return nil, err
		}

		health := 100
		if session.HealthPercent != nil {
			health = *session.HealthPercent
		}

		payload := map[string]any{
			"session_id":     sid,
			"status":         session.Status,
			"health_percent": health,
		}

		if session.CaseId != nil {
			if caseRes, cErr := h.Clients.Case.GetById(context.Background(), &m.ById{Id: *session.CaseId}); cErr == nil && caseRes != nil && caseRes.InitialVitals != nil {
				vitals, alarm := liveVitals(*caseRes.InitialVitals, health)
				payload["vitals"] = vitals
				payload["alarm"] = alarm // ""|"warning"|"critical"|"flatline" - front shu bo'yicha tovush/rang tanlaydi
			}
		}
		return payload, nil
	})

	hub.Register(conn)
	defer hub.Unregister(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// liveVitals - health_percent'ga qarab boshlang'ich vitallarni (CaseVitals)
// tibbiy mantiqqa mos tomon suradi (sog'liq tushganda HR/RR oshadi, SpO2
// tushadi) + jonli monitor hissi uchun kichik jitter qo'shadi. alarm -
// front tomon ogohlantiruvchi rang/tovushni tanlashi uchun daraja qaytaradi.
func liveVitals(base m.CaseVitals, healthPercent int) (m.CaseVitals, string) {
	if healthPercent < 0 {
		healthPercent = 0
	}
	if healthPercent > 100 {
		healthPercent = 100
	}
	severity := float64(100-healthPercent) / 100 // 0 (sog'lom) .. 1 (og'ir)

	jitter := func(spread int) int {
		if spread <= 0 {
			return 0
		}
		return rand.Intn(2*spread+1) - spread
	}

	v := base
	v.HR = clampInt(base.HR+int(severity*45)+jitter(2), 30, 220)
	v.RR = clampInt(base.RR+int(severity*14)+jitter(1), 0, 50)
	v.SpO2 = clampInt(base.SpO2-int(severity*20)+jitter(1), 40, 100)
	v.Temp = base.Temp + severity*1.2

	// BP - "systolic/diastolic" matn ko'rinishida, severity oshganda pasayadi (shok holati).
	if sys, dia, ok := parseBP(base.BP); ok {
		sys = clampInt(sys-int(severity*35)+jitter(2), 40, 220)
		dia = clampInt(dia-int(severity*20)+jitter(1), 30, 140)
		v.BP = strconv.Itoa(sys) + "/" + strconv.Itoa(dia)
	}

	alarm := ""
	switch {
	case healthPercent <= 0:
		alarm = "flatline"
	case severity >= 0.7:
		alarm = "critical"
	case severity >= 0.35:
		alarm = "warning"
	}
	return v, alarm
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func parseBP(bp string) (sys, dia int, ok bool) {
	parts := strings.SplitN(bp, "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	s, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	d, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return s, d, true
}
