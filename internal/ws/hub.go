// Package ws - Simulyatsiya real-vaqt monitori (BACKEND_STRUCTURE.md 4-bo'lim:
// vital parametrlar oqimi, monitor tovush signallari). Har bir simulyatsiya
// sessiyasi uchun bitta Hub yaratiladi; ulanish ochilganda tick-goroutine
// vitallarni yuboradi. Hozircha tick ichidagi vitallarni hisoblash mantiqi
// TODO - health_percent asosida statik qiymat yuboradi.
package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// VitalsFetcher - hub'ga har tick'da joriy holatni (health_percent va h.k.)
// beruvchi funksiya turi; service qatlamidan inject qilinadi.
type VitalsFetcher func(sessionId string) (map[string]any, error)

// Hub - bitta simulyatsiya sessiyasi uchun ulanishlar to'plami.
type Hub struct {
	sessionId string
	mu        sync.Mutex
	conns     map[*websocket.Conn]struct{}
	fetch     VitalsFetcher
	stop      chan struct{}
}

var (
	hubsMu sync.Mutex
	hubs   = map[string]*Hub{}
)

// GetOrCreateHub - sessionId bo'yicha mavjud hub'ni qaytaradi yoki yangi yaratadi.
func GetOrCreateHub(sessionId string, fetch VitalsFetcher) *Hub {
	hubsMu.Lock()
	defer hubsMu.Unlock()

	if h, ok := hubs[sessionId]; ok {
		return h
	}

	h := &Hub{
		sessionId: sessionId,
		conns:     make(map[*websocket.Conn]struct{}),
		fetch:     fetch,
		stop:      make(chan struct{}),
	}
	hubs[sessionId] = h
	go h.tickLoop()
	return h
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = struct{}{}
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
	conn.Close()

	if len(h.conns) == 0 {
		close(h.stop)
		hubsMu.Lock()
		delete(hubs, h.sessionId)
		hubsMu.Unlock()
	}
}

func (h *Hub) broadcast(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.conns {
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
}

// tickLoop - har soniyada vitallarni push qiladi (BACKEND_STRUCTURE.md 4-bo'lim:
// "har soniyada yangilanuvchi ma'lumotlar"). TODO: health_percent'ga qarab
// HR/BP/SpO2/RR/Temp/GCS'ni haqiqiy tibbiy mantiq bilan hisoblash, alarm/flatline
// tovush signallarini triggerlash.
func (h *Hub) tickLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stop:
			return
		case <-ticker.C:
			if h.fetch == nil {
				continue
			}
			data, err := h.fetch(h.sessionId)
			if err != nil {
				continue
			}
			payload, err := json.Marshal(data)
			if err != nil {
				continue
			}
			h.broadcast(payload)
		}
	}
}
