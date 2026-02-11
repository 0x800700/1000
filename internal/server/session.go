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

	rules := engine.TisyachaPreset()
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
		legal := engine.LegalActions(s.state, player)
		if len(legal) == 0 {
			log.Printf("bot no legal actions: player=%d phase=%v", player, s.state.Round.Phase)
			s.sendError("bot_no_actions", "bot has no legal actions")
			return
		}
		prev := s.state
		action := bot.ChooseAction(s.state, player)
		if err := engine.ApplyAction(&s.state, player, action); err != nil {
			log.Printf("bot action error: player=%d phase=%v action=%v err=%v", player, s.state.Round.Phase, action.Type, err)
			// Phase-aware fallback to avoid stalls
			action = fallbackAction(s.state, player, legal)
			if err2 := engine.ApplyAction(&s.state, player, action); err2 != nil {
				log.Printf("bot fallback error: player=%d phase=%v action=%v err=%v", player, s.state.Round.Phase, action.Type, err2)
				s.sendError("bot_action_failed", "bot action failed")
				return
			}
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

func fallbackAction(state engine.GameState, player int, legal []engine.Action) engine.Action {
	switch state.Round.Phase {
	case engine.PhaseBidding:
		for _, a := range legal {
			if a.Type == engine.ActionPass {
				return a
			}
		}
		minBid := -1
		var pick engine.Action
		for _, a := range legal {
			if a.Type != engine.ActionBid {
				continue
			}
			if minBid == -1 || a.Bid < minBid {
				minBid = a.Bid
				pick = a
			}
		}
		return pick
	case engine.PhaseSnos:
		count := state.Rules.SnosCards
		hand := append([]engine.Card(nil), state.Players[player].Hand...)
		// sort by lowest points then rank strength
		for i := 0; i < len(hand); i++ {
			for j := i + 1; j < len(hand); j++ {
				pi := engine.CardPoints(hand[i].Rank)
				pj := engine.CardPoints(hand[j].Rank)
				if pj < pi || (pj == pi && engine.RankStrength(hand[j].Rank) < engine.RankStrength(hand[i].Rank)) {
					hand[i], hand[j] = hand[j], hand[i]
				}
			}
		}
		if count > len(hand) {
			count = len(hand)
		}
		return engine.Action{Type: engine.ActionSnos, Cards: hand[:count]}
	case engine.PhasePlayTricks:
		lowest := engine.Action{}
		best := -1
		for _, a := range legal {
			if a.Type != engine.ActionPlayCard || a.Card == nil {
				continue
			}
			score := engine.CardPoints(a.Card.Rank)*10 + engine.RankStrength(a.Card.Rank)
			if best == -1 || score < best {
				best = score
				lowest = a
			}
		}
		return lowest
	default:
		if len(legal) > 0 {
			return legal[0]
		}
		return engine.Action{Type: engine.ActionPass}
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
		s.state = engine.NewGame(engine.TisyachaPreset(), 0)
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
	if message != "" {
		log.Printf("ws error: code=%s detail=%s", code, message)
	}
	message = translateErrorMessage(code, message)
	msg := ServerMessage{
		Type:  "error",
		Error: &ErrorView{Code: code, Message: message},
	}
	_ = s.conn.WriteJSON(msg)
}

func translateErrorMessage(code, detail string) string {
	switch code {
	case "bad_request":
		return "Некорректный запрос"
	case "unknown_type":
		return "Неизвестный тип сообщения"
	case "not_started":
		return "Игра ещё не началась"
	case "missing_action_id":
		return "Не указан идентификатор действия"
	case "bad_action":
		return "Некорректное действие"
	case "apply_failed":
		return "Действие невозможно"
	case "bot_no_actions":
		return "Бот не может сделать ход"
	case "bot_action_failed":
		return "Бот не смог выполнить ход"
	default:
		return "Произошла ошибка"
	}
}
