export type Suit = 'C' | 'D' | 'H' | 'S'
export type Rank = '9' | '10' | 'J' | 'Q' | 'K' | 'A'

export type Card = {
  suit: Suit
  rank: Rank
}

export type ActionDTO = {
  type: string
  bid?: number
  suit?: Suit
  card?: Card
  cards?: Card[]
}

export type PlayerView = {
  id: number
  hand?: Card[]
  handCount: number
  roundPts: number
  gameScore: number
  tricks: number
}

export type RoundView = {
  phase: string
  dealer: number
  leader: number
  trump?: Suit
  kittyCount: number
  bidTurn: number
  bidWinner: number
  bidValue: number
  bids?: Record<string, number>
  passed?: Record<string, boolean>
  trickCards: Card[]
  trickOrder: number[]
}

export type GameView = {
  players: PlayerView[]
  round: RoundView
  rules: {
    handSize: number
    kittySize: number
    bidMin: number
    bidStep: number
    maxBid: number
  }
  legalActions: ActionDTO[]
  meta: {
    sessionId: string
    playerId: number
  }
}

export type ServerMessage =
  | { type: 'state'; state: GameView; events?: any[] }
  | { type: 'error'; error: { code: string; message: string } }
