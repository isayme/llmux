import { useState, useEffect } from 'react'
import { api } from '@/services/api'
import type { Provider } from '@/types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table'
import { Server, RefreshCw, ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/Button'

export function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchProviders = async () => {
    setIsLoading(true)
    setError('')
    try {
      const response = await api.getProviders()
      setProviders(response.providers)
    } catch (e) {
      setError('Failed to fetch providers')
      console.error(e)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchProviders()
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Providers</h1>
          <p className="text-[hsl(var(--muted-foreground))]">
            Manage your LLM service providers
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={fetchProviders} disabled={isLoading}>
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

      {providers.length === 0 && !isLoading && !error ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Server className="h-12 w-12 text-[hsl(var(--muted-foreground))] mb-4" />
            <p className="text-lg font-medium">No providers configured</p>
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Add providers in your config.yaml file
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">All Providers</CardTitle>
            <CardDescription>{providers.length} provider(s) configured</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Base URL</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {providers.map((provider) => (
                  <TableRow key={provider.id}>
                    <TableCell className="font-mono text-sm">{provider.id}</TableCell>
                    <TableCell className="font-medium">{provider.name || provider.id}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{provider.type}</Badge>
                    </TableCell>
                    <TableCell className="max-w-xs">
                      <a
                        href={provider.base_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-[hsl(var(--primary))] hover:underline break-all"
                      >
                        {provider.base_url}
                        <ExternalLink className="h-3 w-3 shrink-0" />
                      </a>
                    </TableCell>
                    <TableCell>
                      <Badge variant={provider.enabled ? 'success' : 'destructive'}>
                        {provider.enabled ? 'Enabled' : 'Disabled'}
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
