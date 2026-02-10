package engine

import "testing"

func TestDealDeterministic(t *testing.T) {
	r := ClassicPreset()
	g1 := NewGame(r, 42)
	g2 := NewGame(r, 42)

	DealRound(&g1)
	DealRound(&g2)

	for i := 0; i < r.Players; i++ {
		if len(g1.Players[i].Hand) != r.HandSize {
			t.Fatalf("hand size: got %d", len(g1.Players[i].Hand))
		}
		if len(g2.Players[i].Hand) != r.HandSize {
			t.Fatalf("hand size: got %d", len(g2.Players[i].Hand))
		}
		for c := range g1.Players[i].Hand {
			if g1.Players[i].Hand[c] != g2.Players[i].Hand[c] {
				t.Fatalf("determinism mismatch at player %d card %d", i, c)
			}
		}
	}

	if len(g1.Round.Kitty) != r.KittySize {
		t.Fatalf("kitty size: got %d", len(g1.Round.Kitty))
	}
	if len(g2.Round.Kitty) != r.KittySize {
		t.Fatalf("kitty size: got %d", len(g2.Round.Kitty))
	}
}

func TestDealExhaustsDeck(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	DealRound(&g)

	seen := map[Card]bool{}
	for _, p := range g.Players {
		for _, c := range p.Hand {
			if seen[c] {
				t.Fatalf("duplicate card: %v", c)
			}
			seen[c] = true
		}
	}
	for _, c := range g.Round.Kitty {
		if seen[c] {
			t.Fatalf("duplicate card in kitty: %v", c)
		}
		seen[c] = true
	}
	if len(seen) != len(BuildDeck(r)) {
		t.Fatalf("deck not exhausted: got %d", len(seen))
	}
}
