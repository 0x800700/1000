import { useEffect, useRef } from 'react'
import * as PIXI from 'pixi.js'
import type { Card, GameView } from '../types'
import { createTextureResolver } from './textures'

type Props = {
  state: GameView | null
  legalCardKeys: Set<string>
  onPlayCard: (card: Card) => void
}

type Containers = {
  hand: PIXI.Container
  trick: PIXI.Container
  bots: PIXI.Container
  deck: PIXI.Container
  effects: PIXI.Container
}

export default function PixiTable({ state, legalCardKeys, onPlayCard }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const appRef = useRef<PIXI.Application | null>(null)
  const resolverRef = useRef<ReturnType<typeof createTextureResolver> | null>(null)
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
        resolverRef.current = createTextureResolver(app)

        const hand = new PIXI.Container()
        const trick = new PIXI.Container()
        const bots = new PIXI.Container()
        const deck = new PIXI.Container()
        const effects = new PIXI.Container()
        containersRef.current = { hand, trick, bots, deck, effects }

        app.stage.addChild(deck, bots, trick, hand, effects)
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

    const prev = prevRef.current
    const deal = prev.hand.length === 0 && hand.length > 0

    renderHand(containers.hand, resolver, hand, legalCardKeys, onPlayCard, width, height, deal, deckPos, app)

    if (trick.length === 0 && prev.trick.length > 0) {
      fadeOutTrick(containers.trick, app)
    } else {
      renderTrick(containers.trick, resolver, trick, width, height, app)
    }

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

function renderDeck(container: PIXI.Container, resolver: ReturnType<typeof createTextureResolver>, pos: { x: number; y: number }) {
  container.removeChildren()
  const back = new PIXI.Sprite(resolver.getBackTexture())
  back.anchor.set(0.5)
  back.x = pos.x
  back.y = pos.y
  container.addChild(back)
}

function renderBots(container: PIXI.Container, resolver: ReturnType<typeof createTextureResolver>, state: GameView | null, width: number) {
  container.removeChildren()
  const counts = [
    state?.players?.[1]?.handCount ?? 0,
    state?.players?.[2]?.handCount ?? 0
  ]
  const positions = [
    { x: 140, y: 120 },
    { x: width - 140, y: 120 }
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
      s.scale.set(0.7)
      stack.addChild(s)
    }
    container.addChild(stack)
  }
}

function renderHand(
  container: PIXI.Container,
  resolver: ReturnType<typeof createTextureResolver>,
  hand: Card[],
  legal: Set<string>,
  onPlayCard: (card: Card) => void,
  width: number,
  height: number,
  deal: boolean,
  deckPos: { x: number; y: number },
  app: PIXI.Application
) {
  container.removeChildren()
  const positions = handPositions(hand.length, width, height)
  hand.forEach((card, i) => {
    const cardContainer = new PIXI.Container()
    const sprite = new PIXI.Sprite(resolver.getCardTexture(card))
    sprite.anchor.set(0.5)
    sprite.scale.set(0.95)

    const isLegal = legal.has(cardKey(card))
    if (isLegal) {
      const glow = new PIXI.Graphics()
      glow.lineStyle(2, 0xe0c36a, 0.9)
      glow.drawRoundedRect(-39, -57, 78, 114, 10)
      cardContainer.addChild(glow)
    }

    cardContainer.addChild(sprite)
    const pos = positions[i]
    cardContainer.x = deal ? deckPos.x : pos.x
    cardContainer.y = deal ? deckPos.y : pos.y
    cardContainer.rotation = pos.rotation
    cardContainer.alpha = isLegal ? 1 : 0.45
    cardContainer.eventMode = isLegal ? 'static' : 'none'
    if (isLegal) {
      cardContainer.cursor = 'pointer'
      cardContainer.on('pointertap', () => onPlayCard(card))
    }
    container.addChild(cardContainer)
    if (deal) {
      tween(app, cardContainer, { x: pos.x, y: pos.y }, 260 + i * 15)
    }
  })
}

function renderTrick(
  container: PIXI.Container,
  resolver: ReturnType<typeof createTextureResolver>,
  trick: Card[],
  width: number,
  height: number,
  app: PIXI.Application
) {
  container.removeChildren()
  const center = { x: width / 2, y: height / 2 - 20 }
  const startX = center.x - ((trick.length - 1) * 70) / 2
  trick.forEach((card, i) => {
    const sprite = new PIXI.Sprite(resolver.getCardTexture(card))
    sprite.anchor.set(0.5)
    sprite.x = startX + i * 70
    sprite.y = center.y
    sprite.scale.set(1.2)
    sprite.alpha = 0
    container.addChild(sprite)
    tween(app, sprite, { alpha: 1 }, 140)
  })
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
  resolver: ReturnType<typeof createTextureResolver>,
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
  const startX = width / 2 - (count - 1) * 40 / 2
  for (let i = 0; i < count; i++) {
    const offset = i - (count - 1) / 2
    positions.push({
      x: startX + i * 40,
      y: height - 90 - Math.abs(offset) * 3,
      rotation: (offset * 3 * Math.PI) / 180
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
  const center = { x: width / 2, y: height / 2 - 20 }
  const startX = center.x - ((trick.length - 1) * 70) / 2
  return { x: startX + idx * 70, y: center.y }
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
