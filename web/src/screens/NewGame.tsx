import { useNavigate } from 'react-router-dom'

export default function NewGame() {
  const navigate = useNavigate()
  return (
    <section className="panel">
      <h1>New Game</h1>
      <p>Rules preset: Classic (placeholder)</p>
      <button
        className="primary"
        onClick={() => {
          sessionStorage.setItem('startGame', 'classic')
          navigate('/table')
        }}
      >
        Start
      </button>
    </section>
  )
}
