package engine

import "errors"

type ActionType int

const (
	ActionBid ActionType = iota
	ActionPass
	ActionTakeKitty
	ActionSnos
	ActionPlayCard
	ActionRospis
)

type Action struct {
	Type         ActionType
	Bid          int
	Suit         *Suit
	Card         *Card
	Cards        []Card
	MarriageSuit *Suit
}

func LegalActions(g GameState, player int) []Action {
	// Ordering is deterministic based on rules, bidding increments, and hand order.
	switch g.Round.Phase {
	case PhaseBidding:
		return legalBids(g, player)
	case PhaseKittyTake:
		if player != g.Round.BidWinner {
			return nil
		}
		return []Action{{Type: ActionTakeKitty}}
	case PhaseSnos:
		if player != g.Round.BidWinner {
			return nil
		}
		// Too many combinations; client/bot should choose.
		return []Action{{Type: ActionSnos}}
	case PhasePlayTricks:
		actions := legalPlays(g, player)
		// bidder may declare rospis before any cards played
		if player == g.Round.BidWinner && len(g.Round.TrickCards) == 0 && totalTricks(g) == 0 {
			actions = append(actions, Action{Type: ActionRospis})
		}
		return actions
	default:
		return nil
	}
}

// CurrentPlayer returns the player ID expected to act in the current phase.
func CurrentPlayer(g GameState) (int, bool) {
	switch g.Round.Phase {
	case PhaseBidding:
		return g.Round.BidTurn, true
	case PhaseKittyTake, PhaseSnos:
		if g.Round.BidWinner >= 0 {
			return g.Round.BidWinner, true
		}
		return -1, false
	case PhasePlayTricks:
		if len(g.Round.TrickOrder) == 0 {
			return g.Round.Leader, true
		}
		if len(g.Round.TrickCards) >= len(g.Round.TrickOrder) {
			return -1, false
		}
		return g.Round.TrickOrder[len(g.Round.TrickCards)], true
	default:
		return -1, false
	}
}

func ApplyAction(g *GameState, player int, a Action) error {
	switch g.Round.Phase {
	case PhaseBidding:
		return applyBid(g, player, a)
	case PhaseKittyTake:
		return applyKittyTake(g, player, a)
	case PhaseSnos:
		return applySnos(g, player, a)
	case PhasePlayTricks:
		if a.Type == ActionRospis {
			return applyRospis(g, player, a)
		}
		return applyPlay(g, player, a)
	case PhaseScoreRound:
		scoreRound(g)
		return nil
	default:
		return errors.New("invalid phase")
	}
}

func applyBid(g *GameState, player int, a Action) error {
	if player != g.Round.BidTurn {
		return errors.New("not your turn")
	}
	if g.Round.Passed[player] {
		return errors.New("player already passed")
	}

	switch a.Type {
	case ActionPass:
		g.Round.Passed[player] = true
	case ActionBid:
		if a.Bid < g.Rules.BidMin {
			return errors.New("bid below minimum")
		}
		maxBid := g.Rules.MaxBid
		if maxBid <= 0 {
			maxBid = g.Rules.WinScore
		}
		if a.Bid > maxBid {
			return errors.New("bid above maximum")
		}
		if (a.Bid-g.Rules.BidMin)%g.Rules.BidStep != 0 {
			return errors.New("invalid bid step")
		}
		if a.Bid <= g.Round.BidValue {
			return errors.New("bid not high enough")
		}
		g.Round.BidValue = a.Bid
		g.Round.BidWinner = player
		g.Round.Bids[player] = a.Bid
	default:
		return errors.New("invalid action for bidding")
	}

	// Advance bidding
	active := 0
	for p := 0; p < g.Rules.Players; p++ {
		if !g.Round.Passed[p] {
			active++
		}
	}
	if active == 1 && g.Round.BidWinner >= 0 {
		g.Round.Phase = PhaseKittyTake
		return nil
	}
	if active == 0 {
		// All passed, redeal next round
		g.Round.Dealer = (g.Round.Dealer + 1) % g.Rules.Players
		g.ResetRound()
		return nil
	}

	g.Round.BidTurn = nextBidTurn(g)
	return nil
}

func applyKittyTake(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder takes kitty")
	}
	if a.Type != ActionTakeKitty {
		return errors.New("invalid action for kitty")
	}
	g.Players[player].Hand = append(g.Players[player].Hand, g.Round.Kitty...)
	g.Round.Kitty = nil
	g.Round.Phase = PhaseSnos
	return nil
}

func applySnos(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder makes snos")
	}
	if a.Type != ActionSnos || len(a.Cards) != g.Rules.SnosCards {
		return errors.New("snos requires exact number of cards")
	}

	for _, c := range a.Cards {
		if !removeCard(&g.Players[player].Hand, c) {
			return errors.New("snos card not in hand")
		}
	}
	opponents := orderedOpponents(player, g.Rules.Players)
	for i, c := range a.Cards {
		if i < len(opponents) {
			g.Players[opponents[i]].Hand = append(g.Players[opponents[i]].Hand, c)
		}
	}
	if len(g.Players[player].Hand) != g.Rules.PlayHandSize {
		return errors.New("invalid hand size after snos")
	}
	for _, opp := range opponents {
		if len(g.Players[opp].Hand) != g.Rules.PlayHandSize {
			return errors.New("invalid opponent hand size after snos")
		}
	}
	g.Round.Phase = PhasePlayTricks
	g.Round.Leader = g.Round.BidWinner
	g.Round.TrickCards = nil
	g.Round.TrickOrder = nil
	return nil
}

func applyPlay(g *GameState, player int, a Action) error {
	if a.Type != ActionPlayCard || a.Card == nil {
		return errors.New("invalid play action")
	}
	if len(g.Round.TrickOrder) == 0 {
		g.Round.TrickOrder = buildTrickOrder(g.Round.Leader, g.Rules.Players)
	}
	expected := g.Round.TrickOrder[len(g.Round.TrickCards)]
	if player != expected {
		return errors.New("not your turn to play")
	}
	// Validate legal play (follow suit)
	legal := legalPlays(*g, player)
	if !actionInList(Action{Type: ActionPlayCard, Card: a.Card}, legal) {
		return errors.New("illegal card play")
	}
	if a.MarriageSuit != nil {
		if err := applyMarriage(g, player, *a.Card, *a.MarriageSuit); err != nil {
			return err
		}
	}
	if g.Rules.AceMarriageEnabled && a.Card.Rank == RankA {
		if err := applyAceMarriage(g, player, *a.Card); err != nil {
			return err
		}
	}
	if !removeCard(&g.Players[player].Hand, *a.Card) {
		return errors.New("card not in hand")
	}

	g.Round.TrickCards = append(g.Round.TrickCards, *a.Card)
	if len(g.Round.TrickCards) == g.Rules.Players {
		winner := trickWinner(g.Round.TrickOrder, g.Round.TrickCards, g.Round.Trump)
		g.Players[winner].Tricks = append(g.Players[winner].Tricks, append([]Card(nil), g.Round.TrickCards...))
		g.Round.Leader = winner
		g.Round.TrickCards = nil
		g.Round.TrickOrder = nil

		if len(g.Players[winner].Hand) == 0 {
			g.Round.Phase = PhaseScoreRound
			scoreRound(g)
		}
	}
	return nil
}

func applyRospis(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder can declare rospis")
	}
	if len(g.Round.TrickCards) != 0 || totalTricks(*g) != 0 {
		return errors.New("rospis only before any tricks played")
	}
	if a.Type != ActionRospis {
		return errors.New("invalid rospis action")
	}
	bid := g.Round.BidValue
	if bid == 0 && g.Round.Bids != nil {
		if v, ok := g.Round.Bids[player]; ok {
			bid = v
		}
	}
	g.Players[player].GameScore -= bid
	half := bid / 2
	for i := range g.Players {
		if i == player {
			continue
		}
		g.Players[i].GameScore += half
	}
	g.Round.Dealer = (g.Round.Dealer + 1) % g.Rules.Players
	g.ResetRound()
	return nil
}

func legalBids(g GameState, player int) []Action {
	if player != g.Round.BidTurn || g.Round.Passed[player] {
		return nil
	}
	out := []Action{{Type: ActionPass}}
	maxBid := g.Rules.MaxBid
	if maxBid <= 0 {
		maxBid = g.Rules.WinScore
	}
	for bid := g.Rules.BidMin; bid <= maxBid; bid += g.Rules.BidStep {
		if bid > g.Round.BidValue {
			out = append(out, Action{Type: ActionBid, Bid: bid})
		}
	}
	return out
}

func legalPlays(g GameState, player int) []Action {
	if g.Round.Phase != PhasePlayTricks {
		return nil
	}
	if len(g.Round.TrickOrder) == 0 {
		g.Round.TrickOrder = buildTrickOrder(g.Round.Leader, g.Rules.Players)
	}
	expected := g.Round.TrickOrder[len(g.Round.TrickCards)]
	if player != expected {
		return nil
	}
	hand := g.Players[player].Hand
	if len(hand) == 0 {
		return nil
	}
	// If must follow suit and trick has led suit, restrict.
	if g.Rules.MustFollowSuit && len(g.Round.TrickCards) > 0 {
		led := g.Round.TrickCards[0].Suit
		if hasSuit(hand, led) {
			return cardsToActions(filterBySuit(hand, led), g, player)
		}
	}
	return cardsToActions(hand, g, player)
}

func cardsToActions(cards []Card, g GameState, player int) []Action {
	out := make([]Action, 0, len(cards))
	for i := range cards {
		c := cards[i]
		// default play
		out = append(out, Action{Type: ActionPlayCard, Card: &c})
		// marriage play option
		if canDeclareMarriage(g, player, c) {
			suit := c.Suit
			out = append(out, Action{Type: ActionPlayCard, Card: &c, MarriageSuit: &suit})
		}
	}
	return out
}

func buildTrickOrder(leader, players int) []int {
	order := make([]int, 0, players)
	for i := 0; i < players; i++ {
		order = append(order, (leader+i)%players)
	}
	return order
}

func nextBidTurn(g *GameState) int {
	for i := 1; i <= g.Rules.Players; i++ {
		n := (g.Round.BidTurn + i) % g.Rules.Players
		if !g.Round.Passed[n] {
			return n
		}
	}
	return g.Round.BidTurn
}

func hasSuit(cards []Card, suit Suit) bool {
	for _, c := range cards {
		if c.Suit == suit {
			return true
		}
	}
	return false
}

func filterBySuit(cards []Card, suit Suit) []Card {
	out := []Card{}
	for _, c := range cards {
		if c.Suit == suit {
			out = append(out, c)
		}
	}
	return out
}

func removeCard(hand *[]Card, card Card) bool {
	for i, c := range *hand {
		if c == card {
			*hand = append((*hand)[:i], (*hand)[i+1:]...)
			return true
		}
	}
	return false
}

func actionInList(a Action, list []Action) bool {
	for _, l := range list {
		if a.Type != l.Type {
			continue
		}
		if a.Type == ActionPlayCard && l.Card != nil && a.Card != nil && *a.Card == *l.Card {
			return true
		}
	}
	return false
}

func suitPtr(s Suit) *Suit {
	return &s
}

func orderedOpponents(player int, players int) []int {
	out := []int{}
	for i := 1; i < players; i++ {
		out = append(out, (player+i)%players)
	}
	return out
}

func totalTricks(g GameState) int {
	total := 0
	for _, p := range g.Players {
		total += len(p.Tricks)
	}
	return total
}

func canDeclareMarriage(g GameState, player int, card Card) bool {
	if g.Round.Phase != PhasePlayTricks {
		return false
	}
	if g.Rules.MarriageRequiresTrick && len(g.Players[player].Tricks) == 0 {
		return false
	}
	if card.Rank != RankQ && card.Rank != RankK {
		return false
	}
	if g.Round.DeclaredMarriages == nil {
		return false
	}
	if g.Round.DeclaredMarriages[player] != nil && g.Round.DeclaredMarriages[player][card.Suit] {
		return false
	}
	// must have the pair in hand
	hasQ := false
	hasK := false
	for _, c := range g.Players[player].Hand {
		if c.Suit != card.Suit {
			continue
		}
		if c.Rank == RankQ {
			hasQ = true
		}
		if c.Rank == RankK {
			hasK = true
		}
	}
	return hasQ && hasK
}

func applyMarriage(g *GameState, player int, played Card, suit Suit) error {
	if played.Suit != suit || (played.Rank != RankQ && played.Rank != RankK) {
		return errors.New("marriage must be declared with Q or K of suit")
	}
	if g.Rules.MarriageRequiresTrick && len(g.Players[player].Tricks) == 0 {
		return errors.New("marriage requires at least one trick")
	}
	if g.Round.DeclaredMarriages[player] == nil {
		g.Round.DeclaredMarriages[player] = make(map[Suit]bool)
	}
	if g.Round.DeclaredMarriages[player][suit] {
		return errors.New("marriage already declared")
	}
	// ensure pair exists
	hasQ := false
	hasK := false
	for _, c := range g.Players[player].Hand {
		if c.Suit != suit {
			continue
		}
		if c.Rank == RankQ {
			hasQ = true
		}
		if c.Rank == RankK {
			hasK = true
		}
	}
	if !(hasQ && hasK) {
		return errors.New("marriage requires Q and K in hand")
	}
	g.Round.DeclaredMarriages[player][suit] = true
	g.Players[player].MarriagePts += marriageValue(suit)
	g.Round.Trump = &suit
	return nil
}

func applyAceMarriage(g *GameState, player int, played Card) error {
	if played.Rank != RankA {
		return nil
	}
	if g.Rules.MarriageRequiresTrick && len(g.Players[player].Tricks) == 0 {
		return nil
	}
	if g.Round.DeclaredAceMarriage == nil {
		g.Round.DeclaredAceMarriage = make(map[int]bool)
	}
	if g.Round.DeclaredAceMarriage[player] {
		return nil
	}
	aces := 0
	for _, c := range g.Players[player].Hand {
		if c.Rank == RankA {
			aces++
		}
	}
	if aces < 4 {
		return nil
	}
	g.Round.DeclaredAceMarriage[player] = true
	g.Players[player].MarriagePts += 200
	return nil
}
