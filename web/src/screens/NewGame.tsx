import { useNavigate } from 'react-router-dom'

export default function NewGame() {
  const navigate = useNavigate()
  return (
    <section className="panel">
      <h1>Новая игра</h1>
      <p>Набор правил: tisyacha.ru (по умолчанию)</p>
      <button
        className="primary"
        onClick={() => {
          sessionStorage.setItem('startGame', 'tisyacha')
          navigate('/table')
        }}
      >
        Начать
      </button>
    </section>
  )
}
