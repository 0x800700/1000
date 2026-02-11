import { useEffect, useMemo, useRef, useState } from 'react'
import PixiTable from '../pixi/PixiTable'
import { connect } from '../ws'
import type { ActionDTO, Card, GameView, ServerMessage } from '../types'
import CardView from '../components/Card'

export default function Table() {
  const [state, setState] = useState<GameView | null>(null)
  const [log, setLog] = useState<string[]>([])
  const [discardSelection, setDiscardSelection] = useState<Card[]>([])
  const clientRef = useRef<ReturnType<typeof connect> | null>(null)
  const [wsStatus, setWsStatus] = useState<{ readyState: number; error?: string }>({
    readyState: WebSocket.CONNECTING
  })
  const [lastError, setLastError] = useState<string | null>(null)
  const [selectedBid, setSelectedBid] = useState<number | null>(null)
  const [showDebug, setShowDebug] = useState(false)

  useEffect(() => {
    const client = connect((msg: ServerMessage) => {
      if (msg.type === 'state') {
        setState(msg.state)
        if (msg.events && msg.events.length > 0) {
          setLog((prev) => [...prev, ...msg.events.map((e) => e.type)])
        }
      }
      if (msg.type === 'error') {
        setLog((prev) => [...prev, `Error: ${msg.error.message}`])
        setLastError(msg.error.message)
      }
    }, (status) => {
      setWsStatus(status)
      if (status.error) {
        setLastError(status.error)
      }
    })

    clientRef.current = client
    client.send({ type: 'join_session' })
    const start = sessionStorage.getItem('startGame')
    if (start) {
      client.send({ type: 'start_game', ruleset: start })
      sessionStorage.removeItem('startGame')
    }

    return () => client.close()
  }, [])

  const legalCardKeys = useMemo(() => {
    const set = new Set<string>()
    if (!state) return set
    state.legalActions.forEach((a) => {
      if (a.type === 'play_card' && a.card) {
        set.add(cardKey(a.card))
      }
    })
    return set
  }, [state])

  const hand = state?.players[0].hand ?? []
  const trickCards = state?.round.trickCards ?? []

  function sendActionOnSocket(action: ActionDTO) {
    const actionId = `${Date.now()}-${Math.random().toString(36).slice(2)}`
    const ws = clientRef.current
    if (!ws) return
    ws.send({ type: 'player_action', actionId, action })
  }

  function toggleDiscard(card: Card) {
    const exists = discardSelection.find((c) => cardKey(c) === cardKey(card))
    if (exists) {
      setDiscardSelection((prev) => prev.filter((c) => cardKey(c) !== cardKey(card)))
      return
    }
    setDiscardSelection((prev) => [...prev, card])
  }

  const legalActions = state?.legalActions ?? []
  const legalBids = legalActions.filter((a) => a.type === 'bid')
  const canPass = legalActions.some((a) => a.type === 'pass')
  const canAct = legalActions.length > 0
  const hasAction = (type: string) => legalActions.some((a) => a.type === type)
  const canBid = (value: number | null) => value !== null && legalBids.some((b) => b.bid === value)
  const bidValues = legalBids.map((b) => b.bid ?? 0).filter((b) => b > 0)
  const maxBid = bidValues.length > 0 ? Math.max(...bidValues) : null
  const minNext = bidValues.length > 0 ? Math.min(...bidValues) : null
  const bidStep = state?.rules.bidStep ?? 10
  const currentHighest = state?.round.bidValue ?? 0

  useEffect(() => {
    if (minNext !== null) {
      setSelectedBid((prev) => {
        if (prev === null || prev < minNext) return minNext
        if (maxBid !== null && prev > maxBid) return maxBid
        return prev
      })
    }
  }, [minNext, maxBid])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (state?.round.phase !== 'Bidding') return
      if (e.key.toLowerCase() === 'p' && canPass) {
        sendActionOnSocket({ type: 'pass' })
      }
      if (e.key === 'Enter' && selectedBid && bidValues.includes(selectedBid)) {
        sendActionOnSocket({ type: 'bid', bid: selectedBid })
      }
      if (e.key === 'ArrowUp' || e.key === '+') {
        if (selectedBid === null) return
        const next = Math.min(selectedBid + bidStep, maxBid ?? selectedBid)
        setSelectedBid(next)
      }
      if (e.key === 'ArrowDown' || e.key === '-') {
        if (selectedBid === null) return
        const next = Math.max(selectedBid - bidStep, minNext ?? selectedBid)
        setSelectedBid(next)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [state, canPass, selectedBid, bidStep, bidValues, maxBid, minNext])

  function autoAction() {
    if (!state) return
    const phase = state.round.phase
    if (phase === 'Bidding') {
      if (canBid(minNext)) {
        sendActionOnSocket({ type: 'bid', bid: minNext ?? undefined })
      } else if (canPass) {
        sendActionOnSocket({ type: 'pass' })
      }
      return
    }
    if (phase === 'TrumpSelect') {
      const handCounts = countSuits(hand)
      const bestSuit = maxSuit(handCounts)
      sendActionOnSocket({ type: 'choose_trump', suit: bestSuit })
      return
    }
    if (phase === 'Discard') {
      const count = state.rules.kittySize
      const picked = pickLowestPoints(hand, count)
      sendActionOnSocket({ type: 'discard', cards: picked })
      return
    }
    if (phase === 'PlayTricks') {
      const legal = legalActions
        .filter((a) => a.type === 'play_card' && a.card)
        .map((a) => a.card as Card)
      if (legal.length > 0) {
        const lowest = pickLowestPoints(legal, 1)[0]
        sendActionOnSocket({ type: 'play_card', card: lowest })
      }
      return
    }
  }

  return (
    <section className="table-layout">
      <div className="table-canvas">
        <PixiTable trickCards={trickCards} />
        <div className="status">
          <div className="status-row">
            <span>Connection</span>
            <strong>{readyStateLabel(wsStatus.readyState) === 'OPEN' ? 'Connected' : 'Disconnected'}</strong>
          </div>
          <div className="status-row">
            <span>Phase</span>
            <strong>{state?.round.phase ?? '-'}</strong>
          </div>
          <div className="status-row">
            <span>Turn</span>
            <strong>
              {state?.round.trickOrder?.[state?.round.trickCards?.length ?? 0] ??
                state?.round.bidTurn ??
                '-'}
            </strong>
          </div>
          {lastError && <div className="status-error">{lastError}</div>}
          {import.meta.env.DEV && (
            <button className="secondary ghost" onClick={() => setShowDebug((v) => !v)}>
              {showDebug ? 'Hide debug' : 'Show debug'}
            </button>
          )}
          {showDebug && (
            <div className="debug">
              <div>WS: {readyStateLabel(wsStatus.readyState)}</div>
              <div>Session: {state?.meta.sessionId ?? '-'}</div>
              <div>Player: {state?.meta.playerId ?? 0}</div>
            </div>
          )}
        </div>
        {!state && (
          <div className="loading">
            <div>Loading...</div>
            <button className="secondary" onClick={() => clientRef.current?.send({ type: 'request_state' })}>
              Request state
            </button>
          </div>
        )}
        <div className="table-top">
          <div className="bot-panel left">
            <div className="bot-name">Bot A</div>
            <div className="bot-stats">
              <div>Bid: {state?.round.bids?.['1'] ?? '-'}</div>
              <div>Tricks: {state?.players?.[1]?.tricks ?? 0}</div>
              <div>Score: {state?.players?.[1]?.gameScore ?? 0}</div>
            </div>
          </div>
          <div className="bot-panel right">
            <div className="bot-name">Bot B</div>
            <div className="bot-stats">
              <div>Bid: {state?.round.bids?.['2'] ?? '-'}</div>
              <div>Tricks: {state?.players?.[2]?.tricks ?? 0}</div>
              <div>Score: {state?.players?.[2]?.gameScore ?? 0}</div>
            </div>
          </div>
        </div>
        <div className="action-panel">
          {state?.round.phase === 'Bidding' && (
            <div className="bid-panel">
              <div className="instruction">Bidding: choose a bid or Pass</div>
              <div className="bid-info">
                <div>Highest: {currentHighest || '-'}</div>
                <div>Selected: {selectedBid ?? '-'}</div>
                <div>Min next: {minNext ?? '-'}</div>
                <div>Step: {bidStep}</div>
              </div>
              <div className="bid-actions">
                <button className="primary big" disabled={!canPass} onClick={() => sendActionOnSocket({ type: 'pass' })}>
                  Pass
                </button>
                <button
                  className="primary big"
                  disabled={!canBid(selectedBid)}
                  onClick={() => selectedBid && sendActionOnSocket({ type: 'bid', bid: selectedBid })}
                >
                  Bid
                </button>
                <button
                  className="secondary"
                  disabled={selectedBid === null || maxBid === null || !canBid(Math.min(selectedBid + bidStep, maxBid))}
                  onClick={() =>
                    setSelectedBid((prev) =>
                      prev === null ? minNext : Math.min(prev + bidStep, maxBid)
                    )
                  }
                >
                  +{bidStep}
                </button>
                <button
                  className="secondary"
                  disabled={selectedBid === null || maxBid === null || !canBid(Math.min(selectedBid + 50, maxBid))}
                  onClick={() =>
                    setSelectedBid((prev) =>
                      prev === null ? minNext : Math.min(prev + 50, maxBid)
                    )
                  }
                >
                  +50
                </button>
                <button className="secondary" disabled={maxBid === null} onClick={() => setSelectedBid(maxBid)}>
                  Max
                </button>
                <button className="secondary" disabled={!canAct} onClick={autoAction}>
                  Auto
                </button>
              </div>
            </div>
          )}
          {state?.round.phase === 'TrumpSelect' && (
            <div className="action-row">
              <div className="instruction">Select trump</div>
              {(['C', 'D', 'H', 'S'] as const).map((s) => (
                <button
                  key={s}
                  className="primary"
                  disabled={!hasAction('choose_trump')}
                  onClick={() => sendActionOnSocket({ type: 'choose_trump', suit: s })}
                >
                  Trump {s}
                </button>
              ))}
              <button className="secondary" disabled={!canAct} onClick={autoAction}>
                Auto
              </button>
            </div>
          )}
          {state?.round.phase === 'KittyTake' && (
            <div className="action-row">
              <div className="instruction">Take kitty</div>
              <button
                className="primary"
                disabled={!hasAction('take_kitty')}
                onClick={() => sendActionOnSocket({ type: 'take_kitty' })}
              >
                Take Kitty
              </button>
              <button className="secondary" disabled={!canAct} onClick={autoAction}>
                Auto
              </button>
            </div>
          )}
          {state?.round.phase === 'Discard' && (
            <div className="action-row">
              <div className="instruction">Discard {state?.rules.kittySize ?? 3} cards</div>
              <button
                className="primary"
                disabled={
                  discardSelection.length !== (state?.rules.kittySize ?? 3) ||
                  !hasAction('discard')
                }
                onClick={() => {
                  sendActionOnSocket({ type: 'discard', cards: discardSelection })
                  setDiscardSelection([])
                }}
              >
                Discard {state?.rules.kittySize ?? 3}
              </button>
              <button className="secondary" disabled={!canAct} onClick={autoAction}>
                Auto
              </button>
            </div>
          )}
          {state?.round.phase === 'PlayTricks' && (
            <div className="action-row">
              <div className="instruction">Play a card</div>
              <button className="secondary" disabled={!canAct} onClick={autoAction}>
                Auto
              </button>
            </div>
          )}
        </div>
        <div className="hand-fan">
          {hand.map((c, idx) => {
            const key = cardKey(c)
            const isLegal = legalCardKeys.has(key)
            const isSelected = discardSelection.some((d) => cardKey(d) === key)
            return (
              <CardView
                key={key}
                card={c}
                index={idx}
                total={hand.length}
                isLegal={state?.round.phase === 'Discard' ? true : isLegal}
                isSelected={isSelected}
                onClick={() => {
                  if (state?.round.phase === 'Discard') {
                    toggleDiscard(c)
                  } else if (isLegal) {
                    sendActionOnSocket({ type: 'play_card', card: c })
                  }
                }}
              />
            )
          })}
        </div>
      </div>
      <aside className="side-panel log-panel">
        <h2>Event Log</h2>
        {state && (
          <div className="info">
            <div>Phase: {state.round.phase}</div>
            <div>Bid: {state.round.bidValue || '-'}</div>
            <div>Trump: {state.round.trump ?? '-'}</div>
          </div>
        )}
        <ul className="log scroll">
          {log.slice(-10).map((item, idx) => (
            <li key={`${item}-${idx}`}>{item}</li>
          ))}
        </ul>
      </aside>
    </section>
  )
}

function cardKey(c: Card) {
  return `${c.rank}${c.suit}`
}

function readyStateLabel(state: number) {
  switch (state) {
    case WebSocket.CONNECTING:
      return 'CONNECTING'
    case WebSocket.OPEN:
      return 'OPEN'
    case WebSocket.CLOSING:
      return 'CLOSING'
    case WebSocket.CLOSED:
      return 'CLOSED'
    default:
      return String(state)
  }
}

function countSuits(cards: Card[]) {
  const counts: Record<string, number> = { C: 0, D: 0, H: 0, S: 0 }
  cards.forEach((c) => {
    counts[c.suit] = (counts[c.suit] ?? 0) + 1
  })
  return counts
}

function maxSuit(counts: Record<string, number>): 'C' | 'D' | 'H' | 'S' {
  let best: 'C' | 'D' | 'H' | 'S' = 'C'
  let bestVal = -1
  ;(['C', 'D', 'H', 'S'] as const).forEach((s) => {
    if (counts[s] > bestVal) {
      bestVal = counts[s]
      best = s
    }
  })
  return best
}

function pickLowestPoints(cards: Card[], count: number) {
  const score = (c: Card) => {
    switch (c.rank) {
      case 'A':
        return 11
      case '10':
        return 10
      case 'K':
        return 4
      case 'Q':
        return 3
      case 'J':
        return 2
      default:
        return 0
    }
  }
  const sorted = [...cards].sort((a, b) => {
    const sa = score(a)
    const sb = score(b)
    if (sa === sb) return a.rank.localeCompare(b.rank)
    return sa - sb
  })
  return sorted.slice(0, count)
}
