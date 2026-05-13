import { useEffect, useState } from 'react'
import { apiService, type ProviderConfig } from '../services/api'

export default function Providers() {
  const [providers, setProviders] = useState<ProviderConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    apiService.getProviders()
      .then(setProviders)
      .catch(() => setError('Failed to load providers'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--text-secondary)' }}>Loading...</div>
  }

  if (error) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--error)' }}>{error}</div>
  }

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-semibold mb-6" style={{ color: 'var(--text-primary)' }}>Providers</h1>
      {providers.length === 0 ? (
        <div className="p-8 text-center rounded-lg" style={{ backgroundColor: 'var(--bg-secondary)', color: 'var(--text-tertiary)' }}>
          No providers configured
        </div>
      ) : (
        <div className="grid gap-4">
          {providers.map((provider) => (
            <div
              key={provider.id}
              className="p-4 rounded-lg"
              style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border-color)' }}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <span className="font-mono font-medium" style={{ color: 'var(--text-primary)' }}>{provider.name || provider.id}</span>
                </div>
                <span
                  className="text-xs px-2 py-1 rounded-full"
                  style={{
                    backgroundColor: provider.enabled ? 'var(--accent-light)' : 'var(--bg-tertiary)',
                    color: provider.enabled ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                  }}
                >
                  {provider.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <div className="text-sm font-mono" style={{ color: 'var(--text-tertiary)' }}>{provider.base_url}</div>
              <div className="text-xs mt-2" style={{ color: 'var(--text-tertiary)' }}>Type: {provider.type}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}