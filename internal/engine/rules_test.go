package engine

import "testing"

func TestTrickWinnerWithTrump(t *testing.T) {
	trump := SuitSpades
	order := []int{0, 1, 2}
	cards := []Card{
		{Suit: SuitHearts, Rank: RankA},
		{Suit: SuitSpades, Rank: Rank9},
		{Suit: SuitHearts, Rank: Rank10},
	}
	winner := trickWinner(order, cards, &trump)
	if winner != 1 {
		t.Fatalf("expected trump to win trick, got %d", winner)
	}
}

func TestTrickWinnerByRank(t *testing.T) {
	order := []int{0, 1, 2}
	cards := []Card{
		{Suit: SuitClubs, Rank: RankK},
		{Suit: SuitClubs, Rank: Rank10},
		{Suit: SuitClubs, Rank: RankA},
	}
	winner := trickWinner(order, cards, nil)
	if winner != 2 {
		t.Fatalf("expected A to win trick, got %d", winner)
	}
}

func TestScoreRoundContractSuccess(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.BidWinner = 0
	g.Round.BidValue = 20
	g.Round.Bids = map[int]int{0: 20}

	g.Players[0].Tricks = [][]Card{
		{{Suit: SuitHearts, Rank: RankA}, {Suit: SuitSpades, Rank: Rank10}},
	}
	g.Players[1].Tricks = [][]Card{
		{{Suit: SuitDiamonds, Rank: RankK}},
	}
	g.Players[2].Tricks = [][]Card{
		{{Suit: SuitClubs, Rank: RankQ}},
	}

	scoreRound(&g)
	if g.Players[0].GameScore <= 0 {
		t.Fatalf("expected contract player to gain points")
	}
}

func TestScoreRoundContractFail(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.BidWinner = 0
	g.Round.BidValue = 100
	g.Round.Bids = map[int]int{0: 100}

	g.Players[0].Tricks = [][]Card{
		{{Suit: SuitHearts, Rank: Rank9}},
	}
	g.Players[1].Tricks = [][]Card{
		{{Suit: SuitDiamonds, Rank: RankA}},
	}
	g.Players[2].Tricks = [][]Card{
		{{Suit: SuitClubs, Rank: RankA}},
	}

	scoreRound(&g)
	if g.Players[0].GameScore >= 0 {
		t.Fatalf("expected contract player to lose points")
	}
}

func TestGameEndsAtWinScoreNotOnBarrel(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Players[0].GameScore = r.WinScore
	g.Round.BidWinner = 0
	scoreRound(&g)
	if g.Round.Phase != PhaseGameOver {
		t.Fatalf("expected game over when reaching win score")
	}
}

func TestGameDoesNotEndOnBarrel(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Players[0].GameScore = r.WinScore
	g.Players[0].OnBarrel = true
	g.Round.BidWinner = 0
	scoreRound(&g)
	if g.Round.Phase == PhaseGameOver {
		t.Fatalf("game should not end while on barrel")
	}
}

func TestBoltIncrement(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.BidWinner = 0
	scoreRound(&g)
	if g.Players[1].Bolts != 1 || g.Players[2].Bolts != 1 {
		t.Fatalf("expected bolts to increment for players with no tricks")
	}
}

func TestBarrelEntry(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Players[1].GameScore = r.BarrelThreshold
	scoreRound(&g)
	if !g.Players[1].OnBarrel {
		t.Fatalf("expected player to be on barrel at threshold")
	}
}
