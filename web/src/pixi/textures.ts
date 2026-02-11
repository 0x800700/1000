import * as PIXI from 'pixi.js'
import type { Card } from '../types'

type Resolver = {
  getCardTexture: (card: Card) => PIXI.Texture
  getBackTexture: () => PIXI.Texture
  destroy: () => void
}

export function createTextureResolver(app: PIXI.Application): Resolver {
  const cache = new Map<string, PIXI.Texture>()
  const back = createBackTexture(app)

  const suits: Card['suit'][] = ['C', 'D', 'H', 'S']
  const ranks: Card['rank'][] = ['9', '10', 'J', 'Q', 'K', 'A']
  for (const s of suits) {
    for (const r of ranks) {
      const c: Card = { suit: s, rank: r }
      cache.set(cardKey(c), createCardTexture(app, c))
    }
  }

  return {
    getCardTexture: (card) => {
      const key = cardKey(card)
      const tex = cache.get(key)
      if (!tex) {
        const created = createCardTexture(app, card)
        cache.set(key, created)
        return created
      }
      return tex
    },
    getBackTexture: () => back,
    destroy: () => {
      for (const tex of cache.values()) {
        tex.destroy(true)
      }
      back.destroy(true)
    }
  }
}

function cardKey(card: Card) {
  return `${card.rank}${card.suit}`
}

function createCardTexture(app: PIXI.Application, card: Card): PIXI.Texture {
  const width = 78
  const height = 114
  const container = new PIXI.Container()

  const bg = new PIXI.Graphics()
  bg.beginFill(0xf8f1dc)
  bg.lineStyle(2, 0xc9ab5a, 1)
  bg.drawRoundedRect(0, 0, width, height, 10)
  bg.endFill()
  container.addChild(bg)

  const cornerStyle = new PIXI.TextStyle({
    fontFamily: 'Georgia',
    fontSize: 14,
    fill: suitColor(card.suit)
  })
  const corner = new PIXI.Text(`${card.rank}${suitGlyph(card.suit)}`, cornerStyle)
  corner.x = 8
  corner.y = 6
  container.addChild(corner)

  const cornerBottom = new PIXI.Text(`${card.rank}${suitGlyph(card.suit)}`, cornerStyle)
  cornerBottom.anchor.set(1, 1)
  cornerBottom.rotation = Math.PI
  cornerBottom.x = width - 8
  cornerBottom.y = height - 6
  container.addChild(cornerBottom)

  const watermark = new PIXI.Text(suitGlyph(card.suit), {
    fontFamily: 'Georgia',
    fontSize: 44,
    fill: suitColor(card.suit),
    align: 'center'
  })
  watermark.anchor.set(0.5)
  watermark.alpha = 0.15
  watermark.x = width / 2
  watermark.y = height / 2 + 4
  container.addChild(watermark)

  return app.renderer.generateTexture(container, {
    resolution: app.renderer.resolution,
    scaleMode: PIXI.SCALE_MODES.LINEAR
  })
}

function createBackTexture(app: PIXI.Application): PIXI.Texture {
  const width = 78
  const height = 114
  const container = new PIXI.Container()

  const bg = new PIXI.Graphics()
  bg.beginFill(0x1a563b)
  bg.lineStyle(2, 0xc9ab5a, 1)
  bg.drawRoundedRect(0, 0, width, height, 10)
  bg.endFill()
  container.addChild(bg)

  const pattern = new PIXI.Graphics()
  pattern.lineStyle(1, 0xd8c48a, 0.6)
  for (let i = 8; i < width - 8; i += 8) {
    pattern.moveTo(i, 8)
    pattern.lineTo(i, height - 8)
  }
  for (let j = 8; j < height - 8; j += 8) {
    pattern.moveTo(8, j)
    pattern.lineTo(width - 8, j)
  }
  container.addChild(pattern)

  return app.renderer.generateTexture(container, {
    resolution: app.renderer.resolution,
    scaleMode: PIXI.SCALE_MODES.LINEAR
  })
}

function suitGlyph(suit: Card['suit']) {
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

function suitColor(suit: Card['suit']) {
  return suit === 'H' || suit === 'D' ? 0xc62828 : 0x1b1b1b
}
