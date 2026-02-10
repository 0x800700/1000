import { Route, Routes, Link } from 'react-router-dom'
import Home from './screens/Home'
import NewGame from './screens/NewGame'
import Table from './screens/Table'

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">Thousand</div>
        <nav className="nav">
          <Link to="/">Home</Link>
          <Link to="/new">New Game</Link>
          <Link to="/table">Table</Link>
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
  )
}
