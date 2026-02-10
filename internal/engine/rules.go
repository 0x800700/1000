package engine

func rankStrength(r Rank) int {
	switch r {
	case RankA:
		return 6
	case Rank10:
		return 5
	case RankK:
		return 4
	case RankQ:
		return 3
	case RankJ:
		return 2
	case Rank9:
		return 1
	default:
		return 0
	}
}

func cardPoints(r Rank) int {
	switch r {
	case RankA:
		return 11
	case Rank10:
		return 10
	case RankK:
		return 4
	case RankQ:
		return 3
	case RankJ:
		return 2
	case Rank9:
		return 0
	default:
		return 0
	}
}

func trickWinner(order []int, cards []Card, trump *Suit) int {
	if len(order) == 0 || len(cards) == 0 {
		return -1
	}
	leadSuit := cards[0].Suit
	bestIdx := 0
	for i := 1; i < len(cards); i++ {
		c := cards[i]
		best := cards[bestIdx]

		if trump != nil {
			if c.Suit == *trump && best.Suit != *trump {
				bestIdx = i
				continue
			}
			if c.Suit != *trump && best.Suit == *trump {
				continue
			}
		}

		if c.Suit == best.Suit {
			if rankStrength(c.Rank) > rankStrength(best.Rank) {
				bestIdx = i
			}
			continue
		}

		if best.Suit != leadSuit && c.Suit == leadSuit {
			bestIdx = i
		}
	}
	return order[bestIdx]
}

func scoreRound(g *GameState) {
	for i := range g.Players {
		g.Players[i].RoundPts = 0
		for _, trick := range g.Players[i].Tricks {
			for _, c := range trick {
				g.Players[i].RoundPts += cardPoints(c.Rank)
			}
		}
	}

	contract := g.Round.BidWinner
	if contract >= 0 {
		if g.Round.BidValue == 0 && g.Round.Bids != nil {
			if v, ok := g.Round.Bids[contract]; ok {
				g.Round.BidValue = v
			}
		}
		if g.Players[contract].RoundPts >= g.Round.BidValue {
			if g.Rules.ContractScoresAsBid {
				g.Players[contract].GameScore += g.Round.BidValue
			} else {
				g.Players[contract].GameScore += g.Players[contract].RoundPts
			}
		} else {
			if g.Rules.ContractFailPenaltyBid {
				g.Players[contract].GameScore -= g.Round.BidValue
			} else {
				g.Players[contract].GameScore -= g.Players[contract].RoundPts
			}
		}
	}

	for i := range g.Players {
		if i == contract {
			continue
		}
		g.Players[i].GameScore += g.Players[i].RoundPts
	}

	for _, p := range g.Players {
		if p.GameScore >= g.Rules.WinScore {
			g.Round.Phase = PhaseGameOver
			return
		}
	}

	g.Round.Dealer = (g.Round.Dealer + 1) % g.Rules.Players
	g.ResetRound()
}
