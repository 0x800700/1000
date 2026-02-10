package server

import "thousand/internal/engine"

type PlayerView struct {
	ID        int       `json:"id"`
	Hand      []CardDTO `json:"hand,omitempty"`
	HandCount int       `json:"handCount"`
	RoundPts  int       `json:"roundPts"`
	GameScore int       `json:"gameScore"`
	Tricks    int       `json:"tricks"`
}

type RoundView struct {
	Phase      string       `json:"phase"`
	Dealer     int          `json:"dealer"`
	Leader     int          `json:"leader"`
	Trump      *string      `json:"trump,omitempty"`
	KittyCount int          `json:"kittyCount"`
	BidTurn    int          `json:"bidTurn"`
	BidWinner  int          `json:"bidWinner"`
	BidValue   int          `json:"bidValue"`
	Bids       map[int]int  `json:"bids"`
	Passed     map[int]bool `json:"passed"`
	TrickCards []CardDTO    `json:"trickCards"`
	TrickOrder []int        `json:"trickOrder"`
}

type GameView struct {
	Players      []PlayerView `json:"players"`
	Round        RoundView    `json:"round"`
	Rules        RulesView    `json:"rules"`
	LegalActions []ActionDTO  `json:"legalActions"`
}

type RulesView struct {
	HandSize  int `json:"handSize"`
	KittySize int `json:"kittySize"`
	BidMin    int `json:"bidMin"`
	BidStep   int `json:"bidStep"`
}

func BuildGameView(g engine.GameState, viewer int) *GameView {
	players := make([]PlayerView, 0, len(g.Players))
	for i, p := range g.Players {
		view := PlayerView{
			ID:        p.ID,
			HandCount: len(p.Hand),
			RoundPts:  p.RoundPts,
			GameScore: p.GameScore,
			Tricks:    len(p.Tricks),
		}
		if i == viewer {
			for _, c := range p.Hand {
				view.Hand = append(view.Hand, *cardToDTO(c))
			}
		}
		players = append(players, view)
	}
	var trump *string
	if g.Round.Trump != nil {
		s := suitToString(*g.Round.Trump)
		trump = &s
	}
	trickCards := make([]CardDTO, 0, len(g.Round.TrickCards))
	for _, c := range g.Round.TrickCards {
		trickCards = append(trickCards, *cardToDTO(c))
	}
	legal := []ActionDTO{}
	for _, a := range engine.LegalActions(g, viewer) {
		legal = append(legal, ActionFromEngine(a))
	}
	return &GameView{
		Players: players,
		Round: RoundView{
			Phase:      phaseToString(g.Round.Phase),
			Dealer:     g.Round.Dealer,
			Leader:     g.Round.Leader,
			Trump:      trump,
			KittyCount: len(g.Round.Kitty),
			BidTurn:    g.Round.BidTurn,
			BidWinner:  g.Round.BidWinner,
			BidValue:   g.Round.BidValue,
			Bids:       g.Round.Bids,
			Passed:     g.Round.Passed,
			TrickCards: trickCards,
			TrickOrder: g.Round.TrickOrder,
		},
		Rules: RulesView{
			HandSize:  g.Rules.HandSize,
			KittySize: g.Rules.KittySize,
			BidMin:    g.Rules.BidMin,
			BidStep:   g.Rules.BidStep,
		},
		LegalActions: legal,
	}
}

func phaseToString(p engine.Phase) string {
	switch p {
	case engine.PhaseLobby:
		return "Lobby"
	case engine.PhaseDeal:
		return "Deal"
	case engine.PhaseBidding:
		return "Bidding"
	case engine.PhaseTrumpSelect:
		return "TrumpSelect"
	case engine.PhaseKittyTake:
		return "KittyTake"
	case engine.PhaseDiscard:
		return "Discard"
	case engine.PhasePlayTricks:
		return "PlayTricks"
	case engine.PhaseScoreRound:
		return "ScoreRound"
	case engine.PhaseGameOver:
		return "GameOver"
	default:
		return "Unknown"
	}
}
