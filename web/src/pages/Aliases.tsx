import { useEffect, useState } from 'react'
import { apiService, type ModelAlias } from '../services/api'

export default function Aliases() {
  const [aliases, setAliases] = useState<Record<string, ModelAlias>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    apiService.getAliases()
      .then((data) => setAliases(data.aliases))
      .catch(() => setError('Failed to load aliases'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--text-secondary)' }}>Loading...</div>
  }

  if (error) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--error)' }}>{error}</div>
  }

  const aliasList = Object.entries(aliases)

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-semibold mb-6" style={{ color: 'var(--text-primary)' }}>Aliases</h1>
      {aliasList.length === 0 ? (
        <div className="p-8 text-center rounded-lg" style={{ backgroundColor: 'var(--bg-secondary)', color: 'var(--text-tertiary)' }}>
          No aliases configured
        </div>
      ) : (
        <div className="grid gap-4">
          {aliasList.map(([name, alias]) => (
            <div
              key={name}
              className="p-4 rounded-lg"
              style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border-color)' }}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono font-medium" style={{ color: 'var(--text-primary)' }}>{name}</span>
                <span
                  className="text-xs px-2 py-1 rounded-full"
                  style={{
                    backgroundColor: alias.enabled ? 'var(--accent-light)' : 'var(--bg-tertiary)',
                    color: alias.enabled ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                  }}
                >
                  {alias.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <div className="flex items-center gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <span style={{ color: 'var(--text-secondary)' }}>provider:</span>
                  <code className="font-mono" style={{ color: 'var(--accent-primary)' }}>{alias.provider}</code>
                </div>
                <div className="flex items-center gap-2">
                  <span style={{ color: 'var(--text-secondary)' }}>model:</span>
                  <code className="font-mono" style={{ color: 'var(--text-tertiary)' }}>{alias.model}</code>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}