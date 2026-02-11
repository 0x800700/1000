package server

import "thousand/internal/engine"

type EventPayload struct {
	Player int       `json:"player"`
	Bid    int       `json:"bid,omitempty"`
	Suit   string    `json:"suit,omitempty"`
	Cards  []CardDTO `json:"cards,omitempty"`
	Trick  int       `json:"trick,omitempty"`
	Points []int     `json:"points,omitempty"`
}

func buildEvents(prev engine.GameState, next engine.GameState, player int, action engine.Action) []Event {
	events := []Event{}
	switch action.Type {
	case engine.ActionBid:
		events = append(events, Event{Type: "bid_made", Data: EventPayload{Player: player, Bid: action.Bid}})
	case engine.ActionPass:
		events = append(events, Event{Type: "bid_passed", Data: EventPayload{Player: player}})
	case engine.ActionChooseTrump:
		if action.Suit != nil {
			events = append(events, Event{Type: "trump_chosen", Data: EventPayload{Player: player, Suit: suitToString(*action.Suit)}})
		}
	case engine.ActionTakeKitty:
		events = append(events, Event{Type: "kitty_taken", Data: EventPayload{Player: player}})
	case engine.ActionDiscard:
		cards := make([]CardDTO, 0, len(action.Cards))
		for _, c := range action.Cards {
			cards = append(cards, cardToDTO(c))
		}
		events = append(events, Event{Type: "discarded", Data: EventPayload{Player: player, Cards: cards}})
	case engine.ActionPlayCard:
		if action.Card != nil {
			events = append(events, Event{Type: "card_played", Data: EventPayload{Player: player, Cards: []CardDTO{cardToDTO(*action.Card)}}})
		}
	}

	// Trick won
	for i := range next.Players {
		if len(next.Players[i].Tricks) > len(prev.Players[i].Tricks) {
			events = append(events, Event{Type: "trick_won", Data: EventPayload{Player: i}})
		}
	}
	// Round scored
	if prev.Round.Phase != next.Round.Phase && next.Round.Phase == engine.PhaseDeal {
		points := make([]int, 0, len(prev.Players))
		for _, p := range prev.Players {
			points = append(points, p.RoundPts)
		}
		events = append(events, Event{Type: "round_scored", Data: EventPayload{Points: points}})
	}
	return events
}
