import { useState, useEffect } from 'react'
import { api } from '@/services/api'
import type { APIKey } from '@/types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table'
import { Key, RefreshCw, Copy, Check, Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/Button'

export function APIKeysPage() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [copiedKey, setCopiedKey] = useState<string | null>(null)
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set())

  const fetchAPIKeys = async () => {
    setIsLoading(true)
    setError('')
    try {
      const response = await api.getAPIKeys()
      setApiKeys(response.api_keys)
    } catch (e) {
      setError('Failed to fetch API keys')
      console.error(e)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchAPIKeys()
  }, [])

  const copyToClipboard = async (key: string) => {
    await navigator.clipboard.writeText(key)
    setCopiedKey(key)
    setTimeout(() => setCopiedKey(null), 2000)
  }

  const toggleKeyVisibility = (key: string) => {
    setVisibleKeys((prev) => {
      const newSet = new Set(prev)
      if (newSet.has(key)) {
        newSet.delete(key)
      } else {
        newSet.add(key)
      }
      return newSet
    })
  }

  const maskKey = (key: string) => {
    if (key.length <= 8) return '****'
    return key.slice(0, 4) + '****' + key.slice(-4)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
          <p className="text-[hsl(var(--muted-foreground))]">
            Manage API keys for accessing LLM proxy
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={fetchAPIKeys} disabled={isLoading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {error && (
        <Card className="border-[hsl(var(--destructive))]">
          <CardContent className="pt-6">
            <p className="text-[hsl(var(--destructive))]">{error}</p>
          </CardContent>
        </Card>
      )}

      {apiKeys.length === 0 && !isLoading && !error ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Key className="h-12 w-12 text-[hsl(var(--muted-foreground))] mb-4" />
            <p className="text-lg font-medium">No API keys configured</p>
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Add API keys in your config.yaml file
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">All API Keys</CardTitle>
            <CardDescription>{apiKeys.length} API key(s) configured</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.map((apiKey) => (
                  <TableRow key={apiKey.key}>
                    <TableCell className="font-medium">{apiKey.name || '-'}</TableCell>
                    <TableCell className="font-mono text-sm max-w-xs">
                      <span className="flex items-center gap-1 break-all">
                        <span>{visibleKeys.has(apiKey.key) ? apiKey.key : maskKey(apiKey.key)}</span>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0 shrink-0"
                          onClick={() => toggleKeyVisibility(apiKey.key)}
                        >
                          {visibleKeys.has(apiKey.key) ? (
                            <EyeOff className="h-3.5 w-3.5" />
                          ) : (
                            <Eye className="h-3.5 w-3.5" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0 shrink-0"
                          onClick={() => copyToClipboard(apiKey.key)}
                        >
                          {copiedKey === apiKey.key ? (
                            <Check className="h-3.5 w-3.5 text-[hsl(var(--success))]" />
                          ) : (
                            <Copy className="h-3.5 w-3.5" />
                          )}
                        </Button>
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant={apiKey.enabled ? 'success' : 'destructive'}>
                        {apiKey.enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
