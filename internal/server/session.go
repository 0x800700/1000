package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"thousand/internal/bots"
	"thousand/internal/engine"
)

func generateSessionID() string {
	return time.Now().Format("20060102150405")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Session struct {
	mu         sync.Mutex
	id         string
	state      engine.GameState
	started    bool
	actionIds  map[string]bool
	conn       *websocket.Conn
	botPlayers map[int]bots.Bot
}

var (
	sessionOnce sync.Once
	sessionInst *Session
)

func GetSession() *Session {
	sessionOnce.Do(func() {
		sessionInst = &Session{
			id:         generateSessionID(),
			actionIds:  map[string]bool{},
			botPlayers: map[int]bots.Bot{},
		}
	})
	return sessionInst
}

func (s *Session) HandleConnection(conn *websocket.Conn) {
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			s.sendError("bad_request", "invalid json")
			continue
		}
		s.handleMessage(msg)
	}
}

type ClientMessage struct {
	Type      string     `json:"type"`
	ActionId  string     `json:"actionId,omitempty"`
	Action    *ActionDTO `json:"action,omitempty"`
	Ruleset   string     `json:"ruleset,omitempty"`
	RequestId string     `json:"requestId,omitempty"`
}

type ServerMessage struct {
	Type   string     `json:"type"`
	State  *GameView  `json:"state,omitempty"`
	Events []Event    `json:"events,omitempty"`
	Error  *ErrorView `json:"error,omitempty"`
}

type ErrorView struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

func (s *Session) handleMessage(msg ClientMessage) {
	switch msg.Type {
	case "join_session":
		s.sendState(nil)
	case "start_game":
		s.startGame(msg.Ruleset)
	case "request_state":
		s.sendState(nil)
	case "player_action":
		s.applyAction(msg.ActionId, msg.Action)
	default:
		s.sendError("unknown_type", "unknown message type")
	}
}

func (s *Session) startGame(ruleset string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := engine.ClassicPreset()
	s.state = engine.NewGame(rules, time.Now().UnixNano())
	engine.DealRound(&s.state)
	s.started = true
	s.actionIds = map[string]bool{}
	s.botPlayers = map[int]bots.Bot{
		1: bots.NewEasy(s.state.Seed + 1),
		2: bots.NewNormal(s.state.Seed + 2),
	}
	s.sendStateLocked(nil)
	s.botAutoPlayLocked()
}

func (s *Session) applyAction(actionId string, dto *ActionDTO) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		s.sendError("not_started", "game not started")
		return
	}
	if actionId == "" {
		s.sendError("missing_action_id", "actionId required")
		return
	}
	if s.actionIds[actionId] {
		s.sendStateLocked(nil)
		return
	}
	s.actionIds[actionId] = true

	prev := s.state
	action, err := dto.ToEngine()
	if err != nil {
		s.sendError("bad_action", err.Error())
		return
	}
	player := 0
	if err := engine.ApplyAction(&s.state, player, action); err != nil {
		s.sendError("apply_failed", err.Error())
		return
	}
	s.ensureDealLocked()
	events := buildEvents(prev, s.state, player, action)
	s.sendStateLocked(events)
	s.botAutoPlayLocked()
}

func (s *Session) botAutoPlayLocked() {
	for {
		player, ok := engine.CurrentPlayer(s.state)
		if !ok {
			return
		}
		bot, isBot := s.botPlayers[player]
		if !isBot {
			return
		}
		prev := s.state
		action := bot.ChooseAction(s.state, player)
		if err := engine.ApplyAction(&s.state, player, action); err != nil {
			log.Printf("bot action error: %v", err)
			return
		}
		s.ensureDealLocked()
		events := buildEvents(prev, s.state, player, action)
		s.sendStateLocked(events)
	}
}

func (s *Session) ensureDealLocked() {
	if s.state.Round.Phase == engine.PhaseDeal && !s.state.Round.HandsDealt {
		engine.DealRound(&s.state)
	}
}

func (s *Session) sendState(events []Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendStateLocked(events)
}

func (s *Session) sendStateLocked(events []Event) {
	if s.conn == nil {
		return
	}
	if !s.started {
		s.state = engine.NewGame(engine.ClassicPreset(), 0)
	}
	msg := ServerMessage{
		Type:   "state",
		State:  BuildGameView(s.state, 0, s.id),
		Events: events,
	}
	_ = s.conn.WriteJSON(msg)
}

func (s *Session) sendError(code, message string) {
	if s.conn == nil {
		return
	}
	msg := ServerMessage{
		Type:  "error",
		Error: &ErrorView{Code: code, Message: message},
	}
	_ = s.conn.WriteJSON(msg)
}
