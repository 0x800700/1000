package server

import (
	"errors"

	"thousand/internal/engine"
)

type CardDTO struct {
	Suit string `json:"suit"`
	Rank string `json:"rank"`
}

type ActionDTO struct {
	Type  string    `json:"type"`
	Bid   int       `json:"bid,omitempty"`
	Suit  string    `json:"suit,omitempty"`
	Card  *CardDTO  `json:"card,omitempty"`
	Cards []CardDTO `json:"cards,omitempty"`
}

func (a *ActionDTO) ToEngine() (engine.Action, error) {
	if a == nil {
		return engine.Action{}, errors.New("action missing")
	}
	switch a.Type {
	case "bid":
		return engine.Action{Type: engine.ActionBid, Bid: a.Bid}, nil
	case "pass":
		return engine.Action{Type: engine.ActionPass}, nil
	case "choose_trump":
		s, err := parseSuit(a.Suit)
		if err != nil {
			return engine.Action{}, err
		}
		return engine.Action{Type: engine.ActionChooseTrump, Suit: &s}, nil
	case "take_kitty":
		return engine.Action{Type: engine.ActionTakeKitty}, nil
	case "discard":
		if len(a.Cards) == 0 {
			return engine.Action{}, errors.New("discard cards required")
		}
		cards := make([]engine.Card, 0, len(a.Cards))
		for _, c := range a.Cards {
			card, err := c.toEngine()
			if err != nil {
				return engine.Action{}, err
			}
			cards = append(cards, card)
		}
		return engine.Action{Type: engine.ActionDiscard, Cards: cards}, nil
	case "play_card":
		if a.Card == nil {
			return engine.Action{}, errors.New("card required")
		}
		card, err := a.Card.toEngine()
		if err != nil {
			return engine.Action{}, err
		}
		return engine.Action{Type: engine.ActionPlayCard, Card: &card}, nil
	default:
		return engine.Action{}, errors.New("unknown action type")
	}
}

func ActionFromEngine(a engine.Action) ActionDTO {
	switch a.Type {
	case engine.ActionBid:
		return ActionDTO{Type: "bid", Bid: a.Bid}
	case engine.ActionPass:
		return ActionDTO{Type: "pass"}
	case engine.ActionChooseTrump:
		if a.Suit == nil {
			return ActionDTO{Type: "choose_trump"}
		}
		return ActionDTO{Type: "choose_trump", Suit: suitToString(*a.Suit)}
	case engine.ActionTakeKitty:
		return ActionDTO{Type: "take_kitty"}
	case engine.ActionDiscard:
		cards := make([]CardDTO, 0, len(a.Cards))
		for _, c := range a.Cards {
			cards = append(cards, cardToDTO(c))
		}
		return ActionDTO{Type: "discard", Cards: cards}
	case engine.ActionPlayCard:
		if a.Card == nil {
			return ActionDTO{Type: "play_card"}
		}
		return ActionDTO{Type: "play_card", Card: cardToDTO(*a.Card)}
	default:
		return ActionDTO{Type: "unknown"}
	}
}

func (c CardDTO) toEngine() (engine.Card, error) {
	s, err := parseSuit(c.Suit)
	if err != nil {
		return engine.Card{}, err
	}
	r, err := parseRank(c.Rank)
	if err != nil {
		return engine.Card{}, err
	}
	return engine.Card{Suit: s, Rank: r}, nil
}

func cardToDTO(c engine.Card) *CardDTO {
	return &CardDTO{Suit: suitToString(c.Suit), Rank: rankToString(c.Rank)}
}

func parseSuit(s string) (engine.Suit, error) {
	switch s {
	case "C":
		return engine.SuitClubs, nil
	case "D":
		return engine.SuitDiamonds, nil
	case "H":
		return engine.SuitHearts, nil
	case "S":
		return engine.SuitSpades, nil
	default:
		return engine.SuitClubs, errors.New("invalid suit")
	}
}

func parseRank(r string) (engine.Rank, error) {
	switch r {
	case "9":
		return engine.Rank9, nil
	case "J":
		return engine.RankJ, nil
	case "Q":
		return engine.RankQ, nil
	case "K":
		return engine.RankK, nil
	case "10":
		return engine.Rank10, nil
	case "A":
		return engine.RankA, nil
	default:
		return engine.Rank9, errors.New("invalid rank")
	}
}

func suitToString(s engine.Suit) string {
	switch s {
	case engine.SuitClubs:
		return "C"
	case engine.SuitDiamonds:
		return "D"
	case engine.SuitHearts:
		return "H"
	case engine.SuitSpades:
		return "S"
	default:
		return "?"
	}
}

func rankToString(r engine.Rank) string {
	switch r {
	case engine.Rank9:
		return "9"
	case engine.RankJ:
		return "J"
	case engine.RankQ:
		return "Q"
	case engine.RankK:
		return "K"
	case engine.Rank10:
		return "10"
	case engine.RankA:
		return "A"
	default:
		return "?"
	}
}
