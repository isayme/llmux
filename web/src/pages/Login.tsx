import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiService } from '../services/api'

export default function Login() {
  const [masterKey, setMasterKey] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const currentTheme = localStorage.getItem('theme') || 'light'
  document.documentElement.setAttribute('data-theme', currentTheme)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    const success = await apiService.login(masterKey)
    if (success) {
      navigate('/providers')
    } else {
      setError('Invalid master key')
    }
    setLoading(false)
  }

  return (
    <div className="min-h-screen flex items-center justify-center" style={{ backgroundColor: 'var(--bg-primary)' }}>
      <div className="w-full max-w-sm p-8 rounded-xl" style={{ backgroundColor: 'var(--bg-secondary)', boxShadow: 'var(--shadow)', border: '1px solid var(--border-color)' }}>
        <div className="text-center mb-8">
          <h1 className="text-2xl font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>LLMux</h1>
          <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>Enter your master key to continue</p>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <input
              type="password"
              value={masterKey}
              onChange={(e) => setMasterKey(e.target.value)}
              className="w-full px-4 py-3 rounded-lg text-sm focus:outline-none focus:ring-2"
              style={{
                backgroundColor: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                color: 'var(--text-primary)',
              }}
              placeholder="Master Key"
              required
            />
          </div>
          {error && (
            <p className="text-sm mb-4 text-center" style={{ color: 'var(--error)' }}>{error}</p>
          )}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 px-4 rounded-lg font-medium text-sm transition-colors"
            style={{
              backgroundColor: 'var(--accent-primary)',
              color: '#fff',
              opacity: loading ? 0.7 : 1,
            }}
          >
            {loading ? 'Verifying...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  )
}