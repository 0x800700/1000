import { useEffect, useRef } from 'react'
import * as PIXI from 'pixi.js'
import type { Card } from '../types'

type Props = {
  trickCards: Card[]
}

export default function PixiTable({ trickCards }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const appRef = useRef<PIXI.Application | null>(null)

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
      })

    return () => {
      destroyed = true
      app.destroy(true, { children: true, texture: true, baseTexture: true })
    }
  }, [])

  useEffect(() => {
    const app = appRef.current
    if (!app) return

    app.stage.removeChildren()

    const ring = new PIXI.Graphics()
    ring.lineStyle(4, 0xd4b35a, 0.8)
    ring.drawRoundedRect(40, 40, app.screen.width - 80, app.screen.height - 120, 24)
    app.stage.addChild(ring)

    const trickGroup = new PIXI.Container()
    const startX = app.screen.width / 2 - (trickCards.length * 52) / 2
    trickCards.forEach((c, i) => {
      const card = drawCard(c)
      card.x = startX + i * 56
      card.y = app.screen.height / 2 - 80
      trickGroup.addChild(card)
    })
    app.stage.addChild(trickGroup)
  }, [trickCards])

  return <div className="pixi-root" ref={containerRef} />
}

function drawCard(c: Card) {
  const g = new PIXI.Graphics()
  g.beginFill(0xf5e6b3)
  g.lineStyle(2, 0xb08a2e)
  g.drawRoundedRect(0, 0, 48, 72, 6)
  g.endFill()

  const text = new PIXI.Text(`${c.rank}${c.suit}`, {
    fill: 0x2b1f0a,
    fontSize: 14,
    fontFamily: 'Georgia'
  })
  text.x = 6
  text.y = 6
  g.addChild(text)
  return g
}
