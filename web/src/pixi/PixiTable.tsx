import { useEffect, useRef } from 'react'
import * as PIXI from 'pixi.js'

export default function PixiTable() {
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const app = new PIXI.Application({
      resizeTo: containerRef.current,
      background: '#0b5d3b',
      antialias: true
    })

    containerRef.current.appendChild(app.view as HTMLCanvasElement)

    const ring = new PIXI.Graphics()
    ring.lineStyle(4, 0xd4b35a, 0.8)
    ring.drawRoundedRect(40, 40, 720, 420, 24)
    app.stage.addChild(ring)

    const title = new PIXI.Text('Table - PixiJS', {
      fill: 0xf5e6b3,
      fontSize: 24,
      fontFamily: 'Georgia'
    })
    title.x = 60
    title.y = 60
    app.stage.addChild(title)

    return () => {
      app.destroy(true, { children: true, texture: true, baseTexture: true })
    }
  }, [])

  return <div className="pixi-root" ref={containerRef} />
}
