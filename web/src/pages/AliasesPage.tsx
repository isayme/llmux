import { useState, useEffect } from 'react'
import { api } from '@/services/api'
import type { Alias } from '@/types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table'
import { GitBranch, RefreshCw, ArrowRight, Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/Button'

export function AliasesPage() {
  const [aliases, setAliases] = useState<Alias[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [copiedAlias, setCopiedAlias] = useState<string | null>(null)

  const fetchAliases = async () => {
    setIsLoading(true)
    setError('')
    try {
      const response = await api.getAliases()
      const aliasesList = Object.values(response.aliases)
      setAliases(aliasesList)
    } catch (e) {
      setError('Failed to fetch aliases')
      console.error(e)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchAliases()
  }, [])

  const copyToClipboard = async (name: string) => {
    await navigator.clipboard.writeText(name)
    setCopiedAlias(name)
    setTimeout(() => setCopiedAlias(null), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Aliases</h1>
          <p className="text-[hsl(var(--muted-foreground))]">
            Model aliases for easy access to different models
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={fetchAliases} disabled={isLoading}>
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

      {aliases.length === 0 && !isLoading && !error ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <GitBranch className="h-12 w-12 text-[hsl(var(--muted-foreground))] mb-4" />
            <p className="text-lg font-medium">No aliases configured</p>
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Add aliases in your config.yaml file
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">All Aliases</CardTitle>
            <CardDescription>{aliases.length} alias(es) configured</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Alias Name</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {aliases.map((alias) => (
                  <TableRow key={alias.name}>
                    <TableCell className="font-mono text-sm font-medium">{alias.name}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline">{alias.provider}</Badge>
                        <ArrowRight className="h-4 w-4 text-[hsl(var(--muted-foreground))]" />
                        <span className="font-mono text-sm">{alias.model}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={alias.enabled ? 'success' : 'destructive'}>
                        {alias.enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(alias.name)}
                      >
                        {copiedAlias === alias.name ? (
                          <Check className="h-4 w-4 text-[hsl(var(--success))]" />
                        ) : (
                          <Copy className="h-4 w-4" />
                        )}
                      </Button>
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
