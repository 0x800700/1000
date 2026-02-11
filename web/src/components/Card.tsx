import type { Card as CardType } from '../types'

type Props = {
  card: CardType
  index: number
  total: number
  isLegal: boolean
  isSelected?: boolean
  onClick?: () => void
  size?: 'hand' | 'trick'
}

export default function Card({
  card,
  index,
  total,
  isLegal,
  isSelected,
  onClick,
  size = 'hand'
}: Props) {
  const rotation = size === 'hand' ? (index - (total - 1) / 2) * 3 : 0
  const offset = Math.abs(index - (total - 1) / 2)
  const translateY = size === 'hand' ? Math.min(12, offset * 2) : 0
  const color = card.suit === 'H' || card.suit === 'D' ? '#c62828' : '#1b1b1b'
  return (
    <button
      className={`card ${size} ${isLegal ? 'legal' : 'illegal'} ${isSelected ? 'selected' : ''}`}
      style={{ transform: `translateY(${translateY}px) rotate(${rotation}deg)` }}
      onClick={isLegal ? onClick : undefined}
      disabled={!isLegal}
    >
      <div className="corner top-left" style={{ color }}>
        <span className="rank">{card.rank}</span>
        <span className="suit">{suitGlyph(card.suit)}</span>
      </div>
      <div className="watermark" style={{ color }}>
        {suitGlyph(card.suit)}
      </div>
      <div className="corner bottom-right" style={{ color }}>
        <span className="rank">{card.rank}</span>
        <span className="suit">{suitGlyph(card.suit)}</span>
      </div>
    </button>
  )
}

function suitGlyph(suit: CardType['suit']) {
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
