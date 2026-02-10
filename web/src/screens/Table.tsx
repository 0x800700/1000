import PixiTable from '../pixi/PixiTable'

export default function Table() {
  return (
    <section className="table-layout">
      <div className="table-canvas">
        <PixiTable />
      </div>
      <aside className="side-panel">
        <h2>Event Log</h2>
        <ul className="log">
          <li>Game created</li>
          <li>Awaiting server</li>
        </ul>
      </aside>
    </section>
  )
}
