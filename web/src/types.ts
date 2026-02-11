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
  marriageSuit?: Suit
}

export type PlayerView = {
  id: number
  hand?: Card[]
  handCount: number
  roundPts: number
  gameScore: number
  tricks: number
  bolts: number
  onBarrel: boolean
  barrelAttempts: number
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
  winner: number
  hasWinner: boolean
}

export type GameView = {
  players: PlayerView[]
  round: RoundView
  rules: {
    dealHandSize: number
    playHandSize: number
    kittySize: number
    bidMin: number
    bidStep: number
    maxBid: number
    snosCards: number
    barrelAttempts: number
  }
  legalActions: ActionDTO[]
  effects: {
    dumped: number[]
  }
  meta: {
    sessionId: string
    playerId: number
  }
}

export type ServerMessage =
  | { type: 'state'; state: GameView; events?: any[] }
  | { type: 'error'; error: { code: string; message: string } }
