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

// RankStrength exposes the strength ordering for a rank.
func RankStrength(r Rank) int {
	return rankStrength(r)
}

// CardPoints exposes point value for a rank.
func CardPoints(r Rank) int {
	return cardPoints(r)
}

func marriageValue(s Suit) int {
	switch s {
	case SuitHearts:
		return 100
	case SuitDiamonds:
		return 80
	case SuitClubs:
		return 60
	case SuitSpades:
		return 40
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
	g.LastRoundEffects = RoundEffects{}
	g.LastRoundEffects.Winner = -1
	g.LastRoundPoints = make([]int, len(g.Players))
	for i := range g.Players {
		g.Players[i].RoundPts = 0
		for _, trick := range g.Players[i].Tricks {
			for _, c := range trick {
				g.Players[i].RoundPts += cardPoints(c.Rank)
			}
		}
		g.Players[i].RoundPts += g.Players[i].MarriagePts
		g.LastRoundPoints[i] = g.Players[i].RoundPts
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

	// Bolts
	for i := range g.Players {
		if len(g.Players[i].Tricks) == 0 {
			g.Players[i].Bolts++
			g.LastRoundEffects.Bolts = append(g.LastRoundEffects.Bolts, i)
			if g.Players[i].Bolts >= g.Rules.BoltEvery {
				g.Players[i].GameScore -= g.Rules.BoltPenalty
				g.Players[i].Bolts = 0
				g.LastRoundEffects.BoltPenalties = append(g.LastRoundEffects.BoltPenalties, i)
			}
		}
	}

	// Barrel handling
	prevBarrel := make([]bool, len(g.Players))
	barrelOwner := -1
	for i := range g.Players {
		prevBarrel[i] = g.Players[i].OnBarrel
		if g.Players[i].OnBarrel {
			barrelOwner = i
			break
		}
	}
	for i := range g.Players {
		if g.Players[i].GameScore >= g.Rules.BarrelThreshold && i != barrelOwner {
			if barrelOwner >= 0 {
				g.Players[barrelOwner].OnBarrel = false
				g.Players[barrelOwner].BarrelAttempts = 0
			}
			g.Players[i].OnBarrel = true
			g.Players[i].BarrelAttempts = 0
			barrelOwner = i
		}
	}
	if barrelOwner >= 0 {
		if g.Players[barrelOwner].RoundPts >= g.Rules.BarrelTarget {
			g.Players[barrelOwner].OnBarrel = false
			g.Players[barrelOwner].BarrelAttempts = 0
		} else {
			g.Players[barrelOwner].BarrelAttempts++
			if g.Players[barrelOwner].BarrelAttempts >= g.Rules.BarrelAttempts {
				g.Players[barrelOwner].GameScore -= g.Rules.BoltPenalty
				g.Players[barrelOwner].OnBarrel = false
				g.Players[barrelOwner].BarrelAttempts = 0
				g.LastRoundEffects.BarrelPenalty = append(g.LastRoundEffects.BarrelPenalty, barrelOwner)
			}
		}
	}
	for i := range g.Players {
		if !prevBarrel[i] && g.Players[i].OnBarrel {
			g.LastRoundEffects.BarrelEnter = append(g.LastRoundEffects.BarrelEnter, i)
		}
		if prevBarrel[i] && !g.Players[i].OnBarrel {
			g.LastRoundEffects.BarrelExit = append(g.LastRoundEffects.BarrelExit, i)
		}
	}

	// Dump (reset to 0)
	for i := range g.Players {
		if g.Players[i].GameScore >= g.Rules.DumpThreshold || g.Players[i].GameScore <= g.Rules.DumpNegativeThreshold {
			g.Players[i].GameScore = 0
			g.LastRoundEffects.Dumped = append(g.LastRoundEffects.Dumped, i)
		}
	}

	for i, p := range g.Players {
		if p.GameScore >= g.Rules.WinScore && !p.OnBarrel {
			g.Round.Phase = PhaseGameOver
			g.LastRoundEffects.Winner = i
			g.LastRoundEffects.HasWinner = true
			return
		}
	}

	g.Round.Dealer = (g.Round.Dealer + 1) % g.Rules.Players
	g.ResetRound()
}
