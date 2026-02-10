package engine

import "fmt"

type Suit int

type Rank int

const (
	SuitClubs Suit = iota
	SuitDiamonds
	SuitHearts
	SuitSpades
)

const (
	Rank9 Rank = iota
	RankJ
	RankQ
	RankK
	Rank10
	RankA
)

func (s Suit) String() string {
	switch s {
	case SuitClubs:
		return "C"
	case SuitDiamonds:
		return "D"
	case SuitHearts:
		return "H"
	case SuitSpades:
		return "S"
	default:
		return "?"
	}
}

func (r Rank) String() string {
	switch r {
	case Rank9:
		return "9"
	case RankJ:
		return "J"
	case RankQ:
		return "Q"
	case RankK:
		return "K"
	case Rank10:
		return "10"
	case RankA:
		return "A"
	default:
		return "?"
	}
}

type Card struct {
	Suit Suit
	Rank Rank
}

func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank.String(), c.Suit.String())
}

type Phase int

const (
	PhaseLobby Phase = iota
	PhaseDeal
	PhaseBidding
	PhaseTrumpSelect
	PhaseKittyTake
	PhaseDiscard
	PhasePlayTricks
	PhaseScoreRound
	PhaseGameOver
)

type Rules struct {
	Players               int
	DeckRanks             []Rank
	HandSize              int
	KittySize             int
	BidMin                int
	BidStep               int
	WinScore              int
	MustFollowSuit        bool
	MustTrumpIfVoid       bool
	MustOverTrump         bool
	ContractScoresAsBid   bool
	ContractFailPenaltyBid bool
}

func ClassicPreset() Rules {
	return Rules{
		Players:               3,
		DeckRanks:             []Rank{Rank9, RankJ, RankQ, RankK, Rank10, RankA},
		HandSize:              7,
		KittySize:             3,
		BidMin:                80,
		BidStep:               10,
		WinScore:              1000,
		MustFollowSuit:        true,
		MustTrumpIfVoid:       false,
		MustOverTrump:         false,
		ContractScoresAsBid:   false,
		ContractFailPenaltyBid: true,
	}
}

type PlayerState struct {
	ID        int
	Hand      []Card
	Tricks    [][]Card
	RoundPts  int
	GameScore int
}

type RoundState struct {
	Phase      Phase
	Dealer     int
	Leader     int
	Trump      *Suit
	Kitty      []Card
	HandsDealt bool
	Bids       map[int]int
	BidWinner  int
	BidValue   int
	TrickCards []Card
	TrickOrder []int
}

type GameState struct {
	Rules  Rules
	Seed   int64
	Round  RoundState
	Players []PlayerState
}

func NewGame(r Rules, seed int64) GameState {
	players := make([]PlayerState, r.Players)
	for i := 0; i < r.Players; i++ {
		players[i] = PlayerState{ID: i}
	}

	return GameState{
		Rules: r,
		Seed:  seed,
		Round: RoundState{
			Phase:  PhaseDeal,
			Dealer: 0,
		},
		Players: players,
	}
}

func (g *GameState) ResetRound() {
	g.Round = RoundState{
		Phase:  PhaseDeal,
		Dealer: g.Round.Dealer,
	}
	for i := range g.Players {
		g.Players[i].Hand = nil
		g.Players[i].Tricks = nil
		g.Players[i].RoundPts = 0
	}
}
