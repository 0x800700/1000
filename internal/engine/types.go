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
	PhaseKittyTake
	PhaseSnos
	PhasePlayTricks
	PhaseScoreRound
	PhaseGameOver
)

type Rules struct {
	Players                int
	DeckRanks              []Rank
	DealHandSize           int
	PlayHandSize           int
	KittySize              int
	SnosCards              int
	BidMin                 int
	BidStep                int
	MaxBid                 int
	WinScore               int
	MustFollowSuit         bool
	MustTrumpIfVoid        bool
	MustOverTrump          bool
	ContractScoresAsBid    bool
	ContractFailPenaltyBid bool
	MarriageRequiresTrick  bool
	AceMarriageEnabled     bool
	BarrelThreshold        int
	BarrelTarget           int
	BarrelAttempts         int
	BoltPenalty            int
	BoltEvery              int
	DumpThreshold          int
	DumpNegativeThreshold  int
}

func ClassicPreset() Rules {
	return TisyachaPreset()
}

func TisyachaPreset() Rules {
	return Rules{
		Players:                3,
		DeckRanks:              []Rank{Rank9, RankJ, RankQ, RankK, Rank10, RankA},
		DealHandSize:           7,
		PlayHandSize:           8,
		KittySize:              3,
		SnosCards:              2,
		BidMin:                 80,
		BidStep:                10,
		MaxBid:                 300,
		WinScore:               1000,
		MustFollowSuit:         true,
		MustTrumpIfVoid:        false,
		MustOverTrump:          false,
		ContractScoresAsBid:    false,
		ContractFailPenaltyBid: true,
		MarriageRequiresTrick:  true,
		AceMarriageEnabled:     false,
		BarrelThreshold:        880,
		BarrelTarget:           120,
		BarrelAttempts:         3,
		BoltPenalty:            120,
		BoltEvery:              3,
		DumpThreshold:          555,
		DumpNegativeThreshold:  -555,
	}
}

type PlayerState struct {
	ID             int
	Hand           []Card
	Tricks         [][]Card
	RoundPts       int
	GameScore      int
	MarriagePts    int
	Bolts          int
	OnBarrel       bool
	BarrelAttempts int
}

type RoundState struct {
	Phase               Phase
	Dealer              int
	Leader              int
	Trump               *Suit
	Kitty               []Card
	HandsDealt          bool
	Bids                map[int]int
	Passed              map[int]bool
	BidTurn             int
	BidWinner           int
	BidValue            int
	TrickCards          []Card
	TrickOrder          []int
	DeclaredMarriages   map[int]map[Suit]bool
	DeclaredAceMarriage map[int]bool
}

type GameState struct {
	Rules            Rules
	Seed             int64
	Round            RoundState
	Players          []PlayerState
	LastRoundPoints  []int
	LastRoundEffects RoundEffects
}

type RoundEffects struct {
	Bolts         []int
	BoltPenalties []int
	BarrelEnter   []int
	BarrelExit    []int
	BarrelPenalty []int
	Dumped        []int
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
		g.Players[i].MarriagePts = 0
	}
}
