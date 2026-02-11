import { useEffect, useMemo, useRef, useState } from 'react'
import PixiTable from '../pixi/PixiTable'
import { connect } from '../ws'
import type { ActionDTO, Card, GameView, ServerMessage } from '../types'

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
  const [showHelp, setShowHelp] = useState(false)

  useEffect(() => {
    const client = connect((msg: ServerMessage) => {
      if (msg.type === 'state') {
        setState(msg.state)
        if (msg.events && msg.events.length > 0) {
          setLog((prev) => [...prev, ...msg.events.map(formatEvent)])
        }
      }
      if (msg.type === 'error') {
        const translated = translateError(msg.error?.message)
        setLog((prev) => [...prev, `Ошибка: ${translated}`])
        setLastError(translated)
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

  const marriageMap = useMemo(() => {
    const map = new Map<string, string>()
    if (!state) return map
    state.legalActions.forEach((a) => {
      if (a.type === 'play_card' && a.card && a.marriageSuit) {
        map.set(cardKey(a.card), a.marriageSuit)
      }
    })
    return map
  }, [state])

  const hand = state?.players[0].hand ?? []
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
  const currentTurn =
    state?.round.trickOrder?.[state?.round.trickCards?.length ?? 0] ?? state?.round.bidTurn ?? null
  const botThinking = (id: number) => currentTurn === id
  const instruction = state ? phaseInstruction(state.round.phase) : 'Загрузка...'
  const winnerId = state?.round.hasWinner ? state.round.winner : null
  const dumped = state?.effects.dumped ?? []
  const me = state?.players?.[0]

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

  const connected = wsStatus.readyState === WebSocket.OPEN

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
    if (phase === 'Snos') {
      const count = state.rules.snosCards ?? 2
      const picked = pickLowestPoints(hand, count)
      sendActionOnSocket({ type: 'snos', cards: picked })
      return
    }
    if (phase === 'PlayTricks') {
      if (hasAction('rospis')) {
        sendActionOnSocket({ type: 'rospis' })
        return
      }
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
      <div className="main-col">
        <div className="bot-row">
          <div className={`bot-panel left ${botThinking(1) ? 'is-thinking' : ''}`}>
            <div className="bot-header">
              <div className="bot-avatar">А</div>
              <div className="bot-meta">
                <div className="bot-name">Бот А</div>
                {botThinking(1) && <div className="bot-thinking">думает…</div>}
              </div>
            </div>
            <div className="bot-stats">
              <div className="badge">
                <span>СТАВКА</span>
                <strong>{state?.round.bids?.['1'] ?? '-'}</strong>
              </div>
              <div className="badge">
                <span>ЗАКАЗ</span>
                <strong>{state?.round.bidWinner === 1 ? state?.round.bidValue : '-'}</strong>
              </div>
              <div className="badge">
                <span>ВЗЯТКИ</span>
                <strong>{state?.players?.[1]?.tricks ?? 0}</strong>
              </div>
              <div className="badge">
                <span>СЧЁТ</span>
                <strong>{state?.players?.[1]?.gameScore ?? 0}</strong>
              </div>
            </div>
          </div>
          <div className={`bot-panel right ${botThinking(2) ? 'is-thinking' : ''}`}>
            <div className="bot-header">
              <div className="bot-avatar">Б</div>
              <div className="bot-meta">
                <div className="bot-name">Бот Б</div>
                {botThinking(2) && <div className="bot-thinking">думает…</div>}
              </div>
            </div>
            <div className="bot-stats">
              <div className="badge">
                <span>СТАВКА</span>
                <strong>{state?.round.bids?.['2'] ?? '-'}</strong>
              </div>
              <div className="badge">
                <span>ЗАКАЗ</span>
                <strong>{state?.round.bidWinner === 2 ? state?.round.bidValue : '-'}</strong>
              </div>
              <div className="badge">
                <span>ВЗЯТКИ</span>
                <strong>{state?.players?.[2]?.tricks ?? 0}</strong>
              </div>
              <div className="badge">
                <span>СЧЁТ</span>
                <strong>{state?.players?.[2]?.gameScore ?? 0}</strong>
              </div>
            </div>
          </div>
        </div>
        <div className="pixi-surface">
          <PixiTable
            state={state}
            legalCardKeys={legalCardKeys}
            discardSelection={discardSelection}
            isDiscardPhase={state?.round.phase === 'Snos'}
            onPlayCard={(card) =>
              sendActionOnSocket({
                type: 'play_card',
                card,
                marriageSuit: marriageMap.get(cardKey(card))
              })
            }
            onToggleDiscard={(card) => toggleDiscard(card)}
          />
        </div>
        <div className="action-bar">
          <div className="instruction">{instruction}</div>
          <div className="action-hint">Конец игры: 1000 очков. Бочка — с 880, нужно набрать 120 за 3 попытки.</div>
          {me && (
            <div className="self-score">
              Ваш счёт: <strong>{me.gameScore}</strong> • Взятки: <strong>{me.tricks}</strong> • Очки кона:{' '}
              <strong>{me.roundPts}</strong>
            </div>
          )}
          <div className="action-status">
            {connected ? 'Подключено' : 'Отключено'} • Фаза: {phaseLabel(state?.round.phase)} • Ход игрока:{' '}
            {state
              ? state.round.trickOrder?.[state.round.trickCards?.length ?? 0] ?? state.round.bidTurn ?? '-'
              : '-'}
          </div>
          {state?.round.phase === 'GameOver' && winnerId !== null && winnerId >= 0 && (
            <div className="winner-badge">Победитель: игрок {winnerId}</div>
          )}
          {lastError && <div className="status-error">{lastError}</div>}
          <div className="action-row">
            <button className="secondary" onClick={() => setShowHelp((v) => !v)}>
              {showHelp ? 'Скрыть правила' : 'Правила'}
            </button>
            <button className="secondary" disabled={!canAct} onClick={autoAction}>
              Авто
            </button>
            {!state && (
              <button className="secondary" onClick={() => clientRef.current?.send({ type: 'request_state' })}>
                Запросить состояние
              </button>
            )}
          </div>
          {import.meta.env.DEV && (
            <button className="secondary ghost" onClick={() => setShowDebug((v) => !v)}>
              {showDebug ? 'Скрыть отладку' : 'Показать отладку'}
            </button>
          )}
          {showDebug && (
            <div className="debug">
              <div>Связь: {readyStateLabel(wsStatus.readyState)}</div>
              <div>Сессия: {state?.meta.sessionId ?? '-'}</div>
              <div>Игрок: {state?.meta.playerId ?? 0}</div>
            </div>
          )}
          {state?.round.phase === 'Bidding' && (
            <div className="bid-panel">
              <div className="bid-info">
                <div>Текущая ставка: {currentHighest || '-'}</div>
                <div>Ваша ставка: {selectedBid ?? '-'}</div>
                <div>Мин следующая: {minNext ?? '-'}</div>
                <div>Шаг: {bidStep}</div>
              </div>
              <div className="bid-actions">
                <button className="primary big" disabled={!canPass} onClick={() => sendActionOnSocket({ type: 'pass' })}>
                  Пас
                </button>
                <button
                  className="primary big"
                  disabled={!canBid(selectedBid)}
                  onClick={() => selectedBid && sendActionOnSocket({ type: 'bid', bid: selectedBid })}
                >
                  Ставка
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
                  Макс
                </button>
              </div>
            </div>
          )}
          {state?.round.phase === 'KittyTake' && (
            <div className="action-row">
              <button
                className="primary"
                disabled={!hasAction('take_kitty')}
                onClick={() => sendActionOnSocket({ type: 'take_kitty' })}
              >
                Взять прикуп
              </button>
            </div>
          )}
          {state?.round.phase === 'Snos' && (
            <div className="action-row">
              <button
                className="primary"
                disabled={
                  discardSelection.length !== (state?.rules.snosCards ?? 2) ||
                  !hasAction('snos')
                }
                onClick={() => {
                  sendActionOnSocket({ type: 'snos', cards: discardSelection })
                  setDiscardSelection([])
                }}
              >
                Снос
              </button>
            </div>
          )}
          {state?.round.phase === 'PlayTricks' && (
            <div className="action-row">
              {hasAction('rospis') && (
                <button className="secondary" onClick={() => sendActionOnSocket({ type: 'rospis' })}>
                  Роспись
                </button>
              )}
            </div>
          )}
        </div>
      </div>
      <aside className="side-panel log-panel">
        {showHelp ? (
          <div className="help-panel">
            <h2>Правила</h2>
            <div className="help-phase">{phaseHelp(state?.round.phase)}</div>
            <p>
              <strong>Цель:</strong> первым набрать 1000 очков взятками и марьяжами.
            </p>
            <ul className="help-list">
              <li>Торги: выбирайте ставку или Пас.</li>
              <li>Прикуп: победитель торгов берёт 3 карты.</li>
              <li>Снос: отдайте по одной карте каждому сопернику.</li>
              <li>Ход: следуйте масти, если есть.</li>
              <li>Марьяж: Дама+Король одной масти, даёт очки и делает масть козырем.</li>
              <li>Роспись: можно объявить до первого хода.</li>
              <li>Болт: если не взяли ни одной взятки.</li>
              <li>Бочка и самосвал считаются автоматически по очкам.</li>
              <li>Игра заканчивается, когда игрок набирает 1000 очков вне бочки.</li>
            </ul>
            <p>
              <strong>Ставка</strong> — заказ очков. <strong>Взятки</strong> — количество выигранных взяток.
            </p>
            <p>Нужна помощь? Нажмите <strong>Авто</strong>.</p>
          </div>
        ) : (
          <>
            <h2>Журнал событий</h2>
            {state && (
              <div className="info">
                <div>Фаза: {phaseLabel(state.round.phase)}</div>
                <div>Ставка: {state.round.bidValue || '-'}</div>
                <div>Козырь: {state.round.trump ? suitGlyph(state.round.trump) : '-'}</div>
              </div>
            )}
            {state && (
              <div className="status-panel">
                <h3>Статусы</h3>
                <div className="status-list">
                  {state.players.map((p) => (
                    <div key={p.id} className="status-item">
                      <div className="status-title">
                        Игрок {p.id} • Счёт: {p.gameScore} • Взятки: {p.tricks}
                      </div>
                      <div className="status-badges">
                        {p.onBarrel && (
                          <span className="status-badge gold">
                            Бочка {p.barrelAttempts + 1}/{state.rules.barrelAttempts}
                          </span>
                        )}
                        {p.bolts > 0 && <span className="status-badge">Болты: {p.bolts}</span>}
                        {dumped.includes(p.id) && <span className="status-badge warn">Самосвал</span>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
            <ul className="log scroll">
              {log.slice(-20).map((item, idx) => (
                <li key={`${item}-${idx}`}>{item}</li>
              ))}
            </ul>
          </>
        )}
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
      return 'Подключение'
    case WebSocket.OPEN:
      return 'Открыто'
    case WebSocket.CLOSING:
      return 'Закрывается'
    case WebSocket.CLOSED:
      return 'Закрыто'
    default:
      return String(state)
  }
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

function formatEvent(e: any) {
  const p = e.data?.player
  switch (e.type) {
    case 'bid_made':
      return `Игрок ${p} поставил ${e.data?.bid}`
    case 'bid_passed':
      return `Игрок ${p} пас`
    case 'kitty_taken':
      return `Игрок ${p} взял прикуп`
    case 'snos_made':
      return formatSnos(p, e.data?.transfers ?? [])
    case 'marriage_declared':
      return `Игрок ${p} объявил марьяж ${suitGlyph(e.data?.suit)} (+${e.data?.value ?? 0})`
    case 'ace_marriage_declared':
      return `Игрок ${p} объявил тузовый марьяж (+${e.data?.value ?? 200})`
    case 'card_played':
      return `Игрок ${p} сыграл ${formatCard(e.data?.cards?.[0])}`
    case 'trick_won':
      return `Взятку забрал игрок ${p} (+${e.data?.value ?? 0})`
    case 'round_scored':
      return `${formatRoundScore(e.data?.points ?? [])} • Начинается новый кон`
    case 'rospis_declared':
      return `Игрок ${p} объявил роспись`
    case 'bolt_awarded':
      return `Игрок ${p} получил болт`
    case 'bolt_penalty':
      return `Игрок ${p} получил штраф за болты (-${e.data?.value ?? 0})`
    case 'barrel_enter':
      return `Игрок ${p} сел на бочку`
    case 'barrel_exit':
      return `Игрок ${p} сошёл с бочки`
    case 'barrel_penalty':
      return `Игрок ${p} получил штраф за бочку (-${e.data?.value ?? 0})`
    case 'dump_reset':
      return `Игрок ${p} попал на самосвал — счёт обнулён`
    case 'game_ended':
      return `Игра окончена. Победил игрок ${p}`
    default:
      return 'Неизвестное событие'
  }
}

function formatCard(card?: { rank: string; suit: string }) {
  if (!card) return ''
  return `${rankLabel(card.rank)}${suitGlyph(card.suit)}`
}

function suitGlyph(suit?: string) {
  switch (suit) {
    case 'H':
      return '♥'
    case 'D':
      return '♦'
    case 'C':
      return '♣'
    case 'S':
      return '♠'
    default:
      return suit ?? ''
  }
}

function rankLabel(rank?: string) {
  return rank ?? ''
}

function phaseLabel(phase?: string) {
  switch (phase) {
    case 'Bidding':
      return 'Торги'
    case 'KittyTake':
      return 'Прикуп'
    case 'Snos':
      return 'Снос'
    case 'PlayTricks':
      return 'Ход'
    case 'ScoreRound':
      return 'Подсчёт'
    case 'GameOver':
      return 'Игра окончена'
    default:
      return 'Неизвестно'
  }
}

function phaseHelp(phase?: string) {
  switch (phase) {
    case 'Bidding':
      return 'Торги: выберите ставку или Пас.'
    case 'KittyTake':
      return 'Прикуп: возьмите 3 карты.'
    case 'Snos':
      return 'Снос: отдайте по одной карте каждому сопернику.'
    case 'PlayTricks':
      return 'Ход: выберите карту, следуйте масти.'
    case 'ScoreRound':
      return 'Подсчёт: очки начисляются за взятки и марьяжи.'
    default:
      return 'Следуйте подсказкам внизу.'
  }
}

function phaseInstruction(phase?: string) {
  switch (phase) {
    case 'Bidding':
      return 'Торги: выберите ставку или Пас'
    case 'KittyTake':
      return 'Прикуп: возьмите карты'
    case 'Snos':
      return 'Снос: отдайте по одной карте каждому сопернику'
    case 'PlayTricks':
      return 'Ход: выберите подсвеченную карту'
    case 'ScoreRound':
      return 'Подсчёт кона'
    case 'GameOver':
      return 'Игра окончена'
    default:
      return 'Ожидание'
  }
}

function formatSnos(player: number, transfers: Array<{ to: number; card: { rank: string; suit: string } }>) {
  if (!transfers || transfers.length === 0) {
    return `Игрок ${player} сделал снос`
  }
  const parts = transfers.map((t) => `${formatCard(t.card)} игроку ${t.to}`)
  return `Игрок ${player} сделал снос: отдал ${parts.join(' и ')}`
}

function formatRoundScore(points: number[]) {
  if (!points || points.length === 0) return 'Итог кона: очки не рассчитаны'
  const parts = points.map((p, idx) => `игрок ${idx}: ${p}`)
  return `Итог кона: ${parts.join(', ')}`
}

function translateError(message?: string) {
  if (!message) return 'Неизвестная ошибка'
  if (/[A-Za-z]/.test(message)) return 'Произошла ошибка'
  return message
}
