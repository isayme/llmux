import { useEffect, useState } from 'react'
import { apiService, type ApiKeyConfig } from '../services/api'

export default function ApiKeys() {
  const [apiKeys, setApiKeys] = useState<ApiKeyConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    apiService.getApiKeys()
      .then(setApiKeys)
      .catch(() => setError('Failed to load api keys'))
      .finally(() => setLoading(false))
  }, [])

  const maskKey = (key: string): string => {
    if (key.length <= 8) return key
    return key.slice(0, 4) + '****' + key.slice(-4)
  }

  const handleCopy = (key: string) => {
    navigator.clipboard.writeText(key)
    setCopied(key)
    setTimeout(() => setCopied(null), 2000)
  }

  if (loading) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--text-secondary)' }}>Loading...</div>
  }

  if (error) {
    return <div className="flex items-center justify-center h-32" style={{ color: 'var(--error)' }}>{error}</div>
  }

  return (
    <div className="max-w-4xl">
      <h1 className="text-2xl font-semibold mb-6" style={{ color: 'var(--text-primary)' }}>API Keys</h1>
      {apiKeys.length === 0 ? (
        <div className="p-8 text-center rounded-lg" style={{ backgroundColor: 'var(--bg-secondary)', color: 'var(--text-tertiary)' }}>
          No API keys configured
        </div>
      ) : (
        <div className="grid gap-4">
          {apiKeys.map((apiKey, index) => (
            <div
              key={index}
              className="p-4 rounded-lg"
              style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border-color)' }}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-medium" style={{ color: 'var(--text-primary)' }}>{apiKey.name || 'No Name'}</span>
                <span
                  className="text-xs px-2 py-1 rounded-full"
                  style={{
                    backgroundColor: apiKey.enabled ? 'var(--accent-light)' : 'var(--bg-tertiary)',
                    color: apiKey.enabled ? 'var(--accent-primary)' : 'var(--text-tertiary)',
                  }}
                >
                  {apiKey.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>key:</span>
                <code className="flex-1 text-sm font-mono p-2 rounded" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-tertiary)' }}>
                  {maskKey(apiKey.key)}
                </code>
                <button
                  onClick={() => handleCopy(apiKey.key)}
                  className="px-3 py-2 text-sm rounded transition-colors"
                  style={{ backgroundColor: 'var(--accent-primary)', color: '#fff' }}
                >
                  {copied === apiKey.key ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}