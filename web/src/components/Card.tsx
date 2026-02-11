import type { Card as CardType } from '../types'

type Props = {
  card: CardType
  index: number
  total: number
  isLegal: boolean
  isSelected?: boolean
  onClick?: () => void
}

export default function Card({ card, index, total, isLegal, isSelected, onClick }: Props) {
  const rotation = (index - (total - 1) / 2) * 3
  const offset = Math.abs(index - (total - 1) / 2)
  const translateY = Math.min(12, offset * 2)
  const color = card.suit === 'H' || card.suit === 'D' ? '#c62828' : '#1b1b1b'
  return (
    <button
      className={`card ${isLegal ? 'legal' : 'illegal'} ${isSelected ? 'selected' : ''}`}
      style={{ transform: `translateY(${translateY}px) rotate(${rotation}deg)` }}
      onClick={isLegal ? onClick : undefined}
      disabled={!isLegal}
    >
      <span className="rank" style={{ color }}>
        {card.rank}
      </span>
      <span className="suit" style={{ color }}>
        {suitGlyph(card.suit)}
      </span>
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
