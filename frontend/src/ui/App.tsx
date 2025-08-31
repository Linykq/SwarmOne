import { NavLink, Outlet } from 'react-router-dom'

export default function App() {
  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-20 border-b border-white/10 bg-[rgba(11,18,32,0.7)] backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-xl bg-gradient-to-br from-brand-400 to-fuchsia-400 shadow-soft" />
            <span className="font-semibold tracking-wide">SwarmOne</span>
          </div>
          <nav className="flex items-center gap-2">
            <NavLink to="/" end className={({isActive}) => `pill ${isActive ? 'nav-active' : ''}`}>Home</NavLink>
            <NavLink to="/showcase" className={({isActive}) => `pill ${isActive ? 'nav-active' : ''}`}>Showcase</NavLink>
            <NavLink to="/docs" className={({isActive}) => `pill ${isActive ? 'nav-active' : ''}`}>Docs</NavLink>
            <a className="btn-ghost" href="https://github.com/" target="_blank" rel="noreferrer">GitHub</a>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-8">
        <Outlet />
      </main>

      <footer className="max-w-6xl mx-auto px-4 py-8 text-xs text-zinc-400">
        © {new Date().getFullYear()} SwarmOne · Built with love
      </footer>
    </div>
  )
}
