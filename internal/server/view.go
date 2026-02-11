package server

import "thousand/internal/engine"

type PlayerView struct {
	ID             int       `json:"id"`
	Hand           []CardDTO `json:"hand,omitempty"`
	HandCount      int       `json:"handCount"`
	RoundPts       int       `json:"roundPts"`
	GameScore      int       `json:"gameScore"`
	Tricks         int       `json:"tricks"`
	Bolts          int       `json:"bolts"`
	OnBarrel       bool      `json:"onBarrel"`
	BarrelAttempts int       `json:"barrelAttempts"`
}

type RoundView struct {
	Phase         string       `json:"phase"`
	Dealer        int          `json:"dealer"`
	Leader        int          `json:"leader"`
	Trump         *string      `json:"trump,omitempty"`
	KittyCount    int          `json:"kittyCount"`
	BidTurn       int          `json:"bidTurn"`
	BidWinner     int          `json:"bidWinner"`
	BidValue      int          `json:"bidValue"`
	Bids          map[int]int  `json:"bids"`
	Passed        map[int]bool `json:"passed"`
	TrickCards    []CardDTO    `json:"trickCards"`
	TrickOrder    []int        `json:"trickOrder"`
	Winner        int          `json:"winner"`
	HasWinner     bool         `json:"hasWinner"`
	CurrentPlayer int          `json:"currentPlayer"`
	HasCurrent    bool         `json:"hasCurrent"`
}

type GameView struct {
	Players      []PlayerView `json:"players"`
	Round        RoundView    `json:"round"`
	Rules        RulesView    `json:"rules"`
	LegalActions []ActionDTO  `json:"legalActions"`
	Effects      EffectsView  `json:"effects"`
	Meta         MetaView     `json:"meta"`
}

type RulesView struct {
	DealHandSize   int `json:"dealHandSize"`
	PlayHandSize   int `json:"playHandSize"`
	KittySize      int `json:"kittySize"`
	BidMin         int `json:"bidMin"`
	BidStep        int `json:"bidStep"`
	MaxBid         int `json:"maxBid"`
	SnosCards      int `json:"snosCards"`
	BarrelAttempts int `json:"barrelAttempts"`
}

type MetaView struct {
	SessionID string `json:"sessionId"`
	PlayerID  int    `json:"playerId"`
}

type EffectsView struct {
	Dumped []int `json:"dumped"`
}

func BuildGameView(g engine.GameState, viewer int, sessionID string) *GameView {
	players := make([]PlayerView, 0, len(g.Players))
	for i, p := range g.Players {
		view := PlayerView{
			ID:             p.ID,
			HandCount:      len(p.Hand),
			RoundPts:       p.RoundPts,
			GameScore:      p.GameScore,
			Tricks:         len(p.Tricks),
			Bolts:          p.Bolts,
			OnBarrel:       p.OnBarrel,
			BarrelAttempts: p.BarrelAttempts,
		}
		if i == viewer {
			for _, c := range p.Hand {
				view.Hand = append(view.Hand, cardToDTO(c))
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
		trickCards = append(trickCards, cardToDTO(c))
	}
	legal := []ActionDTO{}
	for _, a := range engine.LegalActions(g, viewer) {
		legal = append(legal, ActionFromEngine(a))
	}
	currentPlayer, hasCurrent := engine.CurrentPlayer(g)
	return &GameView{
		Players: players,
		Round: RoundView{
			Phase:         phaseToString(g.Round.Phase),
			Dealer:        g.Round.Dealer,
			Leader:        g.Round.Leader,
			Trump:         trump,
			KittyCount:    len(g.Round.Kitty),
			BidTurn:       g.Round.BidTurn,
			BidWinner:     g.Round.BidWinner,
			BidValue:      g.Round.BidValue,
			Bids:          g.Round.Bids,
			Passed:        g.Round.Passed,
			TrickCards:    trickCards,
			TrickOrder:    g.Round.TrickOrder,
			Winner:        g.LastRoundEffects.Winner,
			HasWinner:     g.LastRoundEffects.HasWinner,
			CurrentPlayer: currentPlayer,
			HasCurrent:    hasCurrent,
		},
		Rules: RulesView{
			DealHandSize:   g.Rules.DealHandSize,
			PlayHandSize:   g.Rules.PlayHandSize,
			KittySize:      g.Rules.KittySize,
			BidMin:         g.Rules.BidMin,
			BidStep:        g.Rules.BidStep,
			MaxBid:         g.Rules.MaxBid,
			SnosCards:      g.Rules.SnosCards,
			BarrelAttempts: g.Rules.BarrelAttempts,
		},
		LegalActions: legal,
		Effects: EffectsView{
			Dumped: append([]int(nil), g.LastRoundEffects.Dumped...),
		},
		Meta: MetaView{
			SessionID: sessionID,
			PlayerID:  viewer,
		},
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
	case engine.PhaseKittyTake:
		return "KittyTake"
	case engine.PhaseSnos:
		return "Snos"
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
