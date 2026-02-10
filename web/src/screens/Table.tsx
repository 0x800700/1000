import { useEffect, useMemo, useRef, useState } from 'react'
import PixiTable from '../pixi/PixiTable'
import { connect } from '../ws'
import type { ActionDTO, Card, GameView, ServerMessage } from '../types'

export default function Table() {
  const [state, setState] = useState<GameView | null>(null)
  const [log, setLog] = useState<string[]>([])
  const [discardSelection, setDiscardSelection] = useState<Card[]>([])
  const clientRef = useRef<ReturnType<typeof connect> | null>(null)

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

  const legalBids = state?.legalActions.filter((a) => a.type === 'bid') ?? []
  const canPass = state?.legalActions.some((a) => a.type === 'pass')

  return (
    <section className="table-layout">
      <div className="table-canvas">
        <PixiTable hand={hand} trickCards={trickCards} legalCardKeys={legalCardKeys} />
        <div className="hand-bar">
          {hand.map((c) => {
            const key = cardKey(c)
            const isLegal = legalCardKeys.has(key)
            const isSelected = discardSelection.some((d) => cardKey(d) === key)
            return (
              <button
                key={key}
                className={`card-btn ${isLegal ? 'legal' : ''} ${isSelected ? 'selected' : ''}`}
                onClick={() => {
                  if (state?.round.phase === 'Discard') {
                    toggleDiscard(c)
                  } else if (isLegal) {
                    sendActionOnSocket({ type: 'play_card', card: c })
                  }
                }}
              >
                {c.rank}
                {c.suit}
              </button>
            )
          })}
        </div>
      </div>
      <aside className="side-panel">
        <h2>Event Log</h2>
        {state && (
          <div className="info">
            <div>Phase: {state.round.phase}</div>
            <div>Bid: {state.round.bidValue || '-'}</div>
            <div>Trump: {state.round.trump ?? '-'}</div>
          </div>
        )}
        <ul className="log">
          {log.slice(-10).map((item, idx) => (
            <li key={`${item}-${idx}`}>{item}</li>
          ))}
        </ul>
        <div className="actions">
          <h3>Actions</h3>
          {state?.round.phase === 'Bidding' && (
            <div className="action-row">
              {canPass && (
                <button className="secondary" onClick={() => sendActionOnSocket({ type: 'pass' })}>
                  Pass
                </button>
              )}
              {legalBids.map((a) => (
                <button
                  key={`bid-${a.bid}`}
                  className="primary"
                  onClick={() => sendActionOnSocket({ type: 'bid', bid: a.bid })}
                >
                  Bid {a.bid}
                </button>
              ))}
            </div>
          )}
          {state?.round.phase === 'TrumpSelect' && (
            <div className="action-row">
              {(['C', 'D', 'H', 'S'] as const).map((s) => (
                <button key={s} className="primary" onClick={() => sendActionOnSocket({ type: 'choose_trump', suit: s })}>
                  Trump {s}
                </button>
              ))}
            </div>
          )}
          {state?.round.phase === 'KittyTake' && (
            <button className="primary" onClick={() => sendActionOnSocket({ type: 'take_kitty' })}>
              Take Kitty
            </button>
          )}
          {state?.round.phase === 'Discard' && (
            <div className="action-row">
              <button
                className="primary"
                disabled={discardSelection.length !== (state?.rules.kittySize ?? 3)}
                onClick={() => {
                  sendActionOnSocket({ type: 'discard', cards: discardSelection })
                  setDiscardSelection([])
                }}
              >
                Discard {state?.rules.kittySize ?? 3}
              </button>
            </div>
          )}
        </div>
      </aside>
    </section>
  )
}

function cardKey(c: Card) {
  return `${c.rank}${c.suit}`
}
