import { useEffect } from 'react'
import { Route, Routes, Link } from 'react-router-dom'
import Home from './screens/Home'
import NewGame from './screens/NewGame'
import Table from './screens/Table'

export default function App() {
  useEffect(() => {
    const baseWidth = 1280
    const baseHeight = 1000
    const applyScale = () => {
      const scale = Math.min(window.innerWidth / baseWidth, window.innerHeight / baseHeight, 1)
      document.documentElement.style.setProperty('--ui-scale', scale.toString())
    }
    applyScale()
    window.addEventListener('resize', applyScale)
    return () => window.removeEventListener('resize', applyScale)
  }, [])

  return (
    <div className="ui-scale">
      <div className="app">
        <header className="topbar">
          <div className="brand">Тысяча</div>
          <nav className="nav">
            <Link to="/">Главная</Link>
            <Link to="/new">Новая игра</Link>
            <Link to="/table">Стол</Link>
          </nav>
        </header>
        <main className="content">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/new" element={<NewGame />} />
            <Route path="/table" element={<Table />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
