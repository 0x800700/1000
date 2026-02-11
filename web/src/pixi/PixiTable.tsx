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

    const title = new PIXI.Text('Table', {
      fill: 0xe0c36a,
      fontSize: 18,
      fontFamily: 'Georgia'
    })
    title.x = 60
    title.y = 60
    app.stage.addChild(title)
  }, [trickCards])

  return <div className="pixi-root" ref={containerRef} />
}
