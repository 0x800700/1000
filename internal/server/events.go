package server

import "thousand/internal/engine"

type EventPayload struct {
	Player    int            `json:"player"`
	Bid       int            `json:"bid,omitempty"`
	Suit      string         `json:"suit,omitempty"`
	Cards     []CardDTO      `json:"cards,omitempty"`
	Trick     int            `json:"trick,omitempty"`
	Points    []int          `json:"points,omitempty"`
	Value     int            `json:"value,omitempty"`
	Transfers []SnosTransfer `json:"transfers,omitempty"`
}

type SnosTransfer struct {
	To   int     `json:"to"`
	Card CardDTO `json:"card"`
}

func buildEvents(prev engine.GameState, next engine.GameState, player int, action engine.Action) []Event {
	events := []Event{}
	switch action.Type {
	case engine.ActionBid:
		events = append(events, Event{Type: "bid_made", Data: EventPayload{Player: player, Bid: action.Bid}})
	case engine.ActionPass:
		events = append(events, Event{Type: "bid_passed", Data: EventPayload{Player: player}})
	case engine.ActionTakeKitty:
		events = append(events, Event{Type: "kitty_taken", Data: EventPayload{Player: player}})
	case engine.ActionSnos:
		transfers := make([]SnosTransfer, 0, len(action.Cards))
		opponents := orderedOpponents(player, prev.Rules.Players)
		for i, c := range action.Cards {
			if i >= len(opponents) {
				break
			}
			transfers = append(transfers, SnosTransfer{To: opponents[i], Card: cardToDTO(c)})
		}
		events = append(events, Event{Type: "snos_made", Data: EventPayload{Player: player, Transfers: transfers}})
	case engine.ActionPlayCard:
		if action.Card != nil {
			events = append(events, Event{Type: "card_played", Data: EventPayload{Player: player, Cards: []CardDTO{cardToDTO(*action.Card)}}})
		}
		if action.MarriageSuit != nil {
			events = append(events, Event{Type: "marriage_declared", Data: EventPayload{Player: player, Suit: suitToString(*action.MarriageSuit), Value: marriageValueForSuit(*action.MarriageSuit)}})
		}
	case engine.ActionRospis:
		events = append(events, Event{Type: "rospis_declared", Data: EventPayload{Player: player}})
	}

	// Trick won
	for i := range next.Players {
		if len(next.Players[i].Tricks) > len(prev.Players[i].Tricks) {
			points := 0
			last := next.Players[i].Tricks[len(next.Players[i].Tricks)-1]
			for _, c := range last {
				points += engine.CardPoints(c.Rank)
			}
			events = append(events, Event{Type: "trick_won", Data: EventPayload{Player: i, Value: points}})
		}
	}
	// Ace marriage auto-declared
	for i := range next.Players {
		if !prev.Round.DeclaredAceMarriage[i] && next.Round.DeclaredAceMarriage[i] {
			events = append(events, Event{Type: "ace_marriage_declared", Data: EventPayload{Player: i, Value: 200}})
		}
	}
	// Round scored
	if prev.Round.Phase != next.Round.Phase && next.Round.Phase == engine.PhaseDeal {
		points := append([]int(nil), next.LastRoundPoints...)
		events = append(events, Event{Type: "round_scored", Data: EventPayload{Points: points}})
		for _, p := range next.LastRoundEffects.Bolts {
			events = append(events, Event{Type: "bolt_awarded", Data: EventPayload{Player: p}})
		}
		for _, p := range next.LastRoundEffects.BoltPenalties {
			events = append(events, Event{Type: "bolt_penalty", Data: EventPayload{Player: p, Value: next.Rules.BoltPenalty}})
		}
		for _, p := range next.LastRoundEffects.BarrelEnter {
			events = append(events, Event{Type: "barrel_enter", Data: EventPayload{Player: p}})
		}
		for _, p := range next.LastRoundEffects.BarrelExit {
			events = append(events, Event{Type: "barrel_exit", Data: EventPayload{Player: p}})
		}
		for _, p := range next.LastRoundEffects.BarrelPenalty {
			events = append(events, Event{Type: "barrel_penalty", Data: EventPayload{Player: p, Value: next.Rules.BoltPenalty}})
		}
		for _, p := range next.LastRoundEffects.Dumped {
			events = append(events, Event{Type: "dump_reset", Data: EventPayload{Player: p}})
		}
		if next.LastRoundEffects.HasWinner {
			events = append(events, Event{Type: "game_ended", Data: EventPayload{Player: next.LastRoundEffects.Winner}})
		}
	}
	return events
}

func marriageValueForSuit(s engine.Suit) int {
	switch s {
	case engine.SuitHearts:
		return 100
	case engine.SuitDiamonds:
		return 80
	case engine.SuitClubs:
		return 60
	case engine.SuitSpades:
		return 40
	default:
		return 0
	}
}

func orderedOpponents(player int, players int) []int {
	out := []int{}
	for i := 1; i < players; i++ {
		out = append(out, (player+i)%players)
	}
	return out
}
