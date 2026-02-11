import * as PIXI from 'pixi.js'
import type { Card } from '../types'

const CardW = 140
const CardH = 196
const CornerRadius = 14
const BorderWidth = 2
const InnerPadding = 12
const SafeInset = 8
const RankSize = 26
const SuitMiniSize = 22

const PaperTop = 0xfbfbf6
const PaperBottom = 0xf2efe7
const BorderColor = 0xd6d1c6
const InkBlack = 0x1a1a1f
const InkRed = 0xc62828

const BackBase = 0x0f3b2e
const BackBorder = 0xc7a24a

export class CardTextureFactory {
  private cache = new Map<string, PIXI.Texture>()
  private back: PIXI.Texture
  private app: PIXI.Application

  constructor(app: PIXI.Application) {
    this.app = app
    this.back = this.createBackTexture()
    const suits: Card['suit'][] = ['C', 'D', 'H', 'S']
    const ranks: Card['rank'][] = ['9', '10', 'J', 'Q', 'K', 'A']
    for (const s of suits) {
      for (const r of ranks) {
        const c: Card = { suit: s, rank: r }
        this.cache.set(cardKey(c), this.createCardTexture(c))
      }
    }
  }

  getCardTexture(card: Card) {
    const key = cardKey(card)
    const tex = this.cache.get(key)
    if (!tex) {
      const created = this.createCardTexture(card)
      this.cache.set(key, created)
      return created
    }
    return tex
  }

  getBackTexture() {
    return this.back
  }

  destroy() {
    for (const tex of this.cache.values()) {
      tex.destroy(true)
    }
    this.back.destroy(true)
  }

  private createCardTexture(card: Card): PIXI.Texture {
    const container = new PIXI.Container()

    // Shadow
    const shadow = new PIXI.Graphics()
    shadow.beginFill(0x000000, 0.28)
    shadow.drawRoundedRect(0, 8, CardW, CardH, CornerRadius)
    shadow.endFill()
    shadow.filters = [new PIXI.BlurFilter(18)]
    container.addChild(shadow)

    // Paper gradient
    const grad = new PIXI.Graphics()
    grad.beginFill(PaperTop)
    grad.drawRoundedRect(0, 0, CardW, CardH, CornerRadius)
    grad.endFill()
    const gradTex = this.app.renderer.generateTexture(grad)
    const paper = new PIXI.Sprite(gradTex)
    paper.height = CardH
    paper.width = CardW
    const overlay = new PIXI.Graphics()
    overlay.beginFill(PaperBottom, 1)
    overlay.drawRoundedRect(0, CardH / 2, CardW, CardH / 2, CornerRadius)
    overlay.endFill()
    container.addChild(paper, overlay)

    // Border
    const border = new PIXI.Graphics()
    border.lineStyle(BorderWidth, BorderColor, 1)
    border.drawRoundedRect(BorderWidth / 2, BorderWidth / 2, CardW - BorderWidth, CardH - BorderWidth, CornerRadius)
    container.addChild(border)

    const ink = suitColor(card.suit)
    const cornerStyle = new PIXI.TextStyle({
      fontFamily: 'Georgia',
      fontSize: RankSize,
      fill: ink
    })
    const miniStyle = new PIXI.TextStyle({
      fontFamily: 'Georgia',
      fontSize: SuitMiniSize,
      fill: ink
    })

    const rankText = new PIXI.Text(card.rank, cornerStyle)
    rankText.x = InnerPadding
    rankText.y = 10
    container.addChild(rankText)
    const suitMini = new PIXI.Text(suitGlyph(card.suit), miniStyle)
    suitMini.x = InnerPadding
    suitMini.y = 10 + RankSize + 2
    container.addChild(suitMini)

    const rankBottom = new PIXI.Text(card.rank, cornerStyle)
    rankBottom.anchor.set(1, 1)
    rankBottom.rotation = Math.PI
    rankBottom.x = CardW - InnerPadding
    rankBottom.y = CardH - 10
    container.addChild(rankBottom)
    const suitBottom = new PIXI.Text(suitGlyph(card.suit), miniStyle)
    suitBottom.anchor.set(1, 1)
    suitBottom.rotation = Math.PI
    suitBottom.x = CardW - InnerPadding
    suitBottom.y = CardH - (10 + RankSize + 2)
    container.addChild(suitBottom)

    const watermark = new PIXI.Text(suitGlyph(card.suit), {
      fontFamily: 'Georgia',
      fontSize: 92,
      fill: ink
    })
    watermark.anchor.set(0.5)
    watermark.alpha = 0.12
    watermark.x = CardW / 2
    watermark.y = CardH / 2 + 6
    container.addChild(watermark)

    return this.app.renderer.generateTexture(container, {
      resolution: this.app.renderer.resolution,
      scaleMode: PIXI.SCALE_MODES.LINEAR
    })
  }

  private createBackTexture(): PIXI.Texture {
    const container = new PIXI.Container()
    const bg = new PIXI.Graphics()
    bg.beginFill(BackBase)
    bg.drawRoundedRect(0, 0, CardW, CardH, CornerRadius)
    bg.endFill()
    container.addChild(bg)

    const border = new PIXI.Graphics()
    border.lineStyle(BorderWidth, BackBorder, 1)
    border.drawRoundedRect(BorderWidth / 2, BorderWidth / 2, CardW - BorderWidth, CardH - BorderWidth, CornerRadius)
    container.addChild(border)

    const grid = new PIXI.Graphics()
    grid.lineStyle(1, BackBorder, 0.12)
    for (let x = SafeInset; x < CardW - SafeInset; x += 10) {
      grid.moveTo(x, SafeInset)
      grid.lineTo(x + CardH, CardH - SafeInset)
    }
    for (let x = SafeInset; x < CardW - SafeInset; x += 10) {
      grid.moveTo(x, CardH - SafeInset)
      grid.lineTo(x + CardH, SafeInset)
    }
    container.addChild(grid)

    const emblem = new PIXI.Graphics()
    emblem.beginFill(BackBorder, 0.18)
    emblem.drawCircle(CardW / 2, CardH / 2, 26)
    emblem.endFill()
    container.addChild(emblem)

    return this.app.renderer.generateTexture(container, {
      resolution: this.app.renderer.resolution,
      scaleMode: PIXI.SCALE_MODES.LINEAR
    })
  }
}

function cardKey(card: Card) {
  return `${card.rank}${card.suit}`
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
  return suit === 'H' || suit === 'D' ? InkRed : InkBlack
}
