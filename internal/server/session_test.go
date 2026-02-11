package server

import (
	"testing"

	"thousand/internal/engine"
)

func TestFallbackActionIsLegal(t *testing.T) {
	r := engine.TisyachaPreset()
	g := engine.NewGame(r, 1)
	engine.DealRound(&g)

	// Bidding
	player, ok := engine.CurrentPlayer(g)
	if !ok {
		t.Fatalf("no current player in bidding")
	}
	legal := engine.LegalActions(g, player)
	act := fallbackAction(g, player, legal)
	if err := engine.ApplyAction(&g, player, act); err != nil {
		t.Fatalf("fallback action invalid in bidding: %v", err)
	}

	// Force kitty take and snos
	g.Round.Passed = map[int]bool{0: true, 1: true, 2: false}
	g.Round.BidWinner = 2
	g.Round.BidValue = r.BidMin
	g.Round.Phase = engine.PhaseKittyTake
	if err := engine.ApplyAction(&g, 2, engine.Action{Type: engine.ActionTakeKitty}); err != nil {
		t.Fatalf("kitty take failed: %v", err)
	}
	if g.Round.Phase != engine.PhaseSnos {
		t.Fatalf("expected snos phase")
	}
	legal = engine.LegalActions(g, 2)
	act = fallbackAction(g, 2, legal)
	if err := engine.ApplyAction(&g, 2, act); err != nil {
		t.Fatalf("fallback action invalid in snos: %v", err)
	}

	// Play tricks
	player, ok = engine.CurrentPlayer(g)
	if !ok {
		t.Fatalf("no current player in play")
	}
	legal = engine.LegalActions(g, player)
	act = fallbackAction(g, player, legal)
	if err := engine.ApplyAction(&g, player, act); err != nil {
		t.Fatalf("fallback action invalid in play: %v", err)
	}
}
