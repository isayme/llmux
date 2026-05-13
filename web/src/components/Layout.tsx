import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { apiService } from '../services/api'

export default function Layout() {
  const navigate = useNavigate()
  const [currentTheme, setCurrentTheme] = useState(() => localStorage.getItem('theme') || 'light')

  useEffect(() => {
    if (!apiService.checkAuth()) {
      navigate('/login')
    }
  }, [navigate])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', currentTheme)
  }, [currentTheme])

  const handleThemeToggle = () => {
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark'
    localStorage.setItem('theme', newTheme)
    document.documentElement.setAttribute('data-theme', newTheme)
    setCurrentTheme(newTheme)
  }

  const handleLogout = () => {
    apiService.logout()
    navigate('/login')
  }

  const navStyle = ({ isActive }: { isActive: boolean }) => ({
    padding: '0.5rem 1rem',
    borderRadius: '0.375rem',
    fontSize: '0.875rem',
    fontWeight: 500,
    transition: 'all 0.15s',
    backgroundColor: isActive ? 'var(--accent-primary)' : 'transparent',
    color: isActive ? '#fff' : 'var(--text-secondary)',
    textDecoration: 'none',
  })

  return (
    <div className="min-h-screen flex flex-col" style={{ backgroundColor: 'var(--bg-primary)' }}>
      <header className="h-14 flex items-center justify-between px-6" style={{ backgroundColor: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-color)' }}>
        <div className="flex items-center gap-6">
          <h1 className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>LLMux</h1>
          <nav className="flex items-center gap-1">
            <NavLink to="/providers" style={navStyle}>Providers</NavLink>
            <NavLink to="/api-keys" style={navStyle}>API Keys</NavLink>
            <NavLink to="/aliases" style={navStyle}>Aliases</NavLink>
          </nav>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleThemeToggle}
            className="p-2 rounded hover:bg-[var(--bg-tertiary)] transition-colors"
            title={currentTheme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {currentTheme === 'dark' ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
              </svg>
            )}
          </button>
          <button
            onClick={handleLogout}
            className="text-sm px-3 py-1.5 rounded transition-colors"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}
          >
            Logout
          </button>
        </div>
      </header>
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  )
}