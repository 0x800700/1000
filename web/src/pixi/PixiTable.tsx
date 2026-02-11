import { useEffect, useRef } from 'react'
import * as PIXI from 'pixi.js'
import type { Card, GameView } from '../types'
import { CardTextureFactory } from './textures'

type Props = {
  state: GameView | null
  legalCardKeys: Set<string>
  discardSelection: Card[]
  isDiscardPhase: boolean
  onPlayCard: (card: Card) => void
  onToggleDiscard: (card: Card) => void
}

type Containers = {
  hand: PIXI.Container
  trick: PIXI.Container
  trickSlots: PIXI.Container
  trump: PIXI.Container
  bots: PIXI.Container
  deck: PIXI.Container
  effects: PIXI.Container
}

export default function PixiTable({
  state,
  legalCardKeys,
  discardSelection,
  isDiscardPhase,
  onPlayCard,
  onToggleDiscard
}: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const appRef = useRef<PIXI.Application | null>(null)
  const resolverRef = useRef<CardTextureFactory | null>(null)
  const containersRef = useRef<Containers | null>(null)
  const prevRef = useRef<{ hand: Card[]; trick: Card[]; phase?: string }>({ hand: [], trick: [] })

  useEffect(() => {
    if (!containerRef.current) return
    let destroyed = false
    const app = new PIXI.Application()

    app
      .init({
        resizeTo: containerRef.current,
        background: '#0b5d3b',
        antialias: true
      })
      .then(() => {
        if (destroyed) return
        containerRef.current?.appendChild(app.canvas)
        appRef.current = app
        resolverRef.current = new CardTextureFactory(app)

        const hand = new PIXI.Container()
        const trickSlots = new PIXI.Container()
        const trick = new PIXI.Container()
        const trump = new PIXI.Container()
        const bots = new PIXI.Container()
        const deck = new PIXI.Container()
        const effects = new PIXI.Container()
        containersRef.current = { hand, trick, trickSlots, trump, bots, deck, effects }

        app.stage.addChild(deck, bots, trickSlots, trick, hand, trump, effects)
      })

    return () => {
      destroyed = true
      resolverRef.current?.destroy()
      app.destroy(true, { children: true, texture: true, baseTexture: true })
    }
  }, [])

  useEffect(() => {
    const app = appRef.current
    const resolver = resolverRef.current
    const containers = containersRef.current
    if (!app || !resolver || !containers) return

    const width = app.screen.width
    const height = app.screen.height
  const deckPos = { x: width / 2, y: 90 }

    renderDeck(containers.deck, resolver, deckPos)
    renderBots(containers.bots, resolver, state, width)

    const hand = state?.players?.[0]?.hand ?? []
    const trick = state?.round?.trickCards ?? []
    const trump = state?.round?.trump

    const prev = prevRef.current
    const deal = prev.hand.length === 0 && hand.length > 0

    renderHand(
      containers.hand,
      resolver,
      hand,
      legalCardKeys,
      discardSelection,
      onPlayCard,
      onToggleDiscard,
      width,
      height,
      deal,
      deckPos,
      app,
      isDiscardPhase
    )

    renderTrickSlots(containers.trickSlots, width, height)
    if (trick.length === 0 && prev.trick.length > 0) {
      fadeOutTrick(containers.trick, app)
    } else {
      renderTrick(containers.trick, resolver, trick, width, height, app)
    }
    renderTrumpBadge(containers.trump, trump, width, height)

    if (trick.length > prev.trick.length && prev.hand.length > hand.length) {
      const played = trick.find((c) => !prev.trick.some((p) => cardKey(p) === cardKey(c)))
      if (played) {
        const from = prevHandPosition(prev.hand, played, width, height)
        const to = trickPosition(trick, played, width, height)
        animatePlay(containers.effects, resolver, played, from, to, app)
      }
    }

    prevRef.current = { hand, trick, phase: state?.round?.phase }
  }, [state, legalCardKeys, onPlayCard])

  return <div className="pixi-root" ref={containerRef} />
}

function renderDeck(container: PIXI.Container, resolver: CardTextureFactory, pos: { x: number; y: number }) {
  container.removeChildren()
  const back = new PIXI.Sprite(resolver.getBackTexture())
  back.anchor.set(0.5)
  back.x = pos.x
  back.y = pos.y
  container.addChild(back)
}

function renderBots(container: PIXI.Container, resolver: CardTextureFactory, state: GameView | null, width: number) {
  container.removeChildren()
  const counts = [
    state?.players?.[1]?.handCount ?? 0,
    state?.players?.[2]?.handCount ?? 0
  ]
  const positions = [
    { x: 150, y: 120 },
    { x: width - 150, y: 120 }
  ]
  for (let i = 0; i < 2; i++) {
    const stack = new PIXI.Container()
    const back = resolver.getBackTexture()
    const count = Math.min(3, counts[i])
    for (let j = 0; j < count; j++) {
      const s = new PIXI.Sprite(back)
      s.anchor.set(0.5)
      s.x = positions[i].x + j * 4
      s.y = positions[i].y + j * 3
      s.scale.set(0.55)
      stack.addChild(s)
    }
    container.addChild(stack)
  }
}

function renderHand(
  container: PIXI.Container,
  resolver: CardTextureFactory,
  hand: Card[],
  legal: Set<string>,
  discardSelection: Card[],
  onPlayCard: (card: Card) => void,
  onToggleDiscard: (card: Card) => void,
  width: number,
  height: number,
  deal: boolean,
  deckPos: { x: number; y: number },
  app: PIXI.Application,
  isDiscardPhase: boolean
) {
  container.removeChildren()
  const positions = handPositions(hand.length, width, height)
  hand.forEach((card, i) => {
    const cardContainer = new PIXI.Container()
    const sprite = new PIXI.Sprite(resolver.getCardTexture(card))
    sprite.anchor.set(0.5)
    const baseScale = 110 / 140
    sprite.scale.set(1)

    const isLegal = isDiscardPhase ? true : legal.has(cardKey(card))
    if (isLegal) {
      const glow = new PIXI.Graphics()
      glow.lineStyle(3, 0xc7a24a, 0.75)
      glow.drawRoundedRect(-70, -98, 140, 196, 14)
      cardContainer.addChild(glow)
    }
    if (isDiscardPhase && discardSelection.some((c) => cardKey(c) === cardKey(card))) {
      const sel = new PIXI.Graphics()
      sel.lineStyle(3, 0xc7a24a, 0.9)
      sel.drawRoundedRect(-72, -100, 144, 200, 14)
      cardContainer.addChild(sel)
    }

    cardContainer.addChild(sprite)
    const pos = positions[i]
    cardContainer.x = deal ? deckPos.x : pos.x
    cardContainer.y = deal ? deckPos.y : pos.y
    cardContainer.rotation = pos.rotation
    cardContainer.scale.set(baseScale)
    cardContainer.alpha = isLegal ? 1 : 0.45
    cardContainer.eventMode = isLegal ? 'static' : 'none'
    if (isLegal) {
      cardContainer.cursor = 'pointer'
      cardContainer.on('pointertap', () => {
        if (isDiscardPhase) {
          onToggleDiscard(card)
        } else {
          onPlayCard(card)
        }
      })
      cardContainer.on('pointerover', () => {
        cardContainer.y = pos.y - 6
        cardContainer.scale.set(baseScale * 1.02)
      })
      cardContainer.on('pointerout', () => {
        cardContainer.y = pos.y
        cardContainer.scale.set(baseScale)
      })
    }
    container.addChild(cardContainer)
    if (deal) {
      tween(app, cardContainer, { x: pos.x, y: pos.y }, 260 + i * 15)
    }
  })
}

function renderTrick(
  container: PIXI.Container,
  resolver: CardTextureFactory,
  trick: Card[],
  width: number,
  height: number,
  app: PIXI.Application
) {
  container.removeChildren()
  const center = { x: width / 2, y: height / 2 - 6 }
  const slots = trickSlots(trick.length, center)
  trick.forEach((card, i) => {
    const sprite = new PIXI.Sprite(resolver.getCardTexture(card))
    sprite.anchor.set(0.5)
    sprite.x = slots[i].x
    sprite.y = slots[i].y
    sprite.rotation = slots[i].rotation
    sprite.scale.set(1.0)
    sprite.alpha = 0
    container.addChild(sprite)
    tween(app, sprite, { alpha: 1 }, 140)
  })
}

function renderTrickSlots(container: PIXI.Container, width: number, height: number) {
  container.removeChildren()
  const center = { x: width / 2, y: height / 2 - 6 }
  const slots = trickSlots(3, center)
  slots.forEach((s) => {
    const g = new PIXI.Graphics()
    g.lineStyle(2, 0xc7a24a, 0.25)
    g.drawRoundedRect(-70, -98, 140, 196, 14)
    g.x = s.x
    g.y = s.y
    g.rotation = s.rotation
    g.alpha = 0.6
    container.addChild(g)
  })
}

function renderTrumpBadge(container: PIXI.Container, trump: string | undefined, width: number, height: number) {
  container.removeChildren()
  if (!trump) return
  const badge = new PIXI.Graphics()
  badge.beginFill(0x0f3b2e, 0.85)
  badge.lineStyle(2, 0xc7a24a, 0.9)
  badge.drawRoundedRect(0, 0, 86, 34, 10)
  badge.endFill()
  badge.x = width / 2 - 43
  badge.y = height / 2 - 120
  const label = new PIXI.Text(`Trump ${suitGlyph(trump)}`, {
    fontFamily: 'Georgia',
    fontSize: 16,
    fill: 0xf5e6b3
  })
  label.x = badge.x + 12
  label.y = badge.y + 6
  container.addChild(badge, label)
}

function fadeOutTrick(container: PIXI.Container, app: PIXI.Application) {
  container.children.forEach((c) => {
    const sprite = c as PIXI.Sprite
    tween(app, sprite, { alpha: 0, scale: 1.1 }, 180, () => {
      container.removeChildren()
    })
  })
}

function animatePlay(
  effects: PIXI.Container,
  resolver: CardTextureFactory,
  card: Card,
  from: { x: number; y: number },
  to: { x: number; y: number },
  app: PIXI.Application
) {
  const sprite = new PIXI.Sprite(resolver.getCardTexture(card))
  sprite.anchor.set(0.5)
  sprite.x = from.x
  sprite.y = from.y
  sprite.scale.set(1.05)
  effects.addChild(sprite)
  tween(app, sprite, { x: to.x, y: to.y, alpha: 0.2 }, 220, () => {
    effects.removeChild(sprite)
  })
}

function handPositions(count: number, width: number, height: number) {
  const positions: { x: number; y: number; rotation: number }[] = []
  const spacing = 72
  const startX = width / 2 - (count - 1) * spacing / 2
  const maxAngle = 10
  for (let i = 0; i < count; i++) {
    const offset = i - (count - 1) / 2
    positions.push({
      x: startX + i * spacing,
      y: height - 86 - Math.abs(offset) * 3,
      rotation: (offset * maxAngle * 2 * Math.PI) / 180 / Math.max(1, count - 1)
    })
  }
  return positions
}

function prevHandPosition(hand: Card[], card: Card, width: number, height: number) {
  const idx = hand.findIndex((c) => cardKey(c) === cardKey(card))
  if (idx === -1) {
    return { x: width / 2, y: height - 120 }
  }
  const pos = handPositions(hand.length, width, height)[idx]
  return { x: pos.x, y: pos.y }
}

function trickPosition(trick: Card[], card: Card, width: number, height: number) {
  const idx = trick.findIndex((c) => cardKey(c) === cardKey(card))
  const center = { x: width / 2, y: height / 2 - 6 }
  const slots = trickSlots(trick.length, center)
  return { x: slots[idx].x, y: slots[idx].y }
}

function trickSlots(count: number, center: { x: number; y: number }) {
  const positions = [
    { x: center.x - 96, y: center.y + 6, rotation: (-6 * Math.PI) / 180 },
    { x: center.x, y: center.y, rotation: 0 },
    { x: center.x + 96, y: center.y + 6, rotation: (6 * Math.PI) / 180 }
  ]
  if (count === 1) return [positions[1]]
  if (count === 2) return [positions[0], positions[2]]
  return positions
}

function suitGlyph(suit: string) {
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
      return suit
  }
}

function tween(
  app: PIXI.Application,
  sprite: PIXI.DisplayObject,
  to: Partial<{ x: number; y: number; alpha: number; scale: number }>,
  duration: number,
  done?: () => void
) {
  const target = sprite as any
  const from = { x: target.x, y: target.y, alpha: target.alpha ?? 1, scale: target.scale?.x ?? 1 }
  const start = performance.now()
  const update = () => {
    const t = Math.min(1, (performance.now() - start) / duration)
    const ease = t * (2 - t)
    if (to.x !== undefined) target.x = from.x + (to.x - from.x) * ease
    if (to.y !== undefined) target.y = from.y + (to.y - from.y) * ease
    if (to.alpha !== undefined) target.alpha = from.alpha + (to.alpha - from.alpha) * ease
    if (to.scale !== undefined && target.scale) target.scale.set(from.scale + (to.scale - from.scale) * ease)
    if (t >= 1) {
      app.ticker.remove(update)
      if (done) done()
    }
  }
  app.ticker.add(update)
}

function cardKey(card: Card) {
  return `${card.rank}${card.suit}`
}
