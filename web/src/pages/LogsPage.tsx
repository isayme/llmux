import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { FileText, RefreshCw, Trash2, Eye, X, ChevronLeft, ChevronRight, ChevronDown, ChevronRight as ChevronRightIcon, Copy, Check } from 'lucide-react'

interface RequestLog {
  id: number
  request_id: string
  timestamp: string
  model: string
  alias: string
  method: string
  path: string
  client_ip: string
  api_key_id: string
  duration: number
  status: string
  request_body?: number[]
  response_body?: number[]
}

interface ProviderCall {
  id: number
  provider_id: string
  provider_type: string
  model: string
  response_code: number
  duration: number
  is_retry: boolean
  error: string
  request_body?: number[]
  response_header?: Record<string, string[]>
  response_body?: number[]
}

function bytesToString(bytes?: number[]): string {
  if (!bytes || bytes.length === 0) return ''
  return new TextDecoder().decode(new Uint8Array(bytes))
}

function formatJson(str: string): string {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

function CollapsibleJson({ label, data, defaultExpanded = false }: { label: string; data?: number[]; defaultExpanded?: boolean }) {
  const [expanded, setExpanded] = useState(defaultExpanded)
  const [copied, setCopied] = useState(false)
  const str = bytesToString(data)

  if (!str) {
    return (
      <div className="text-sm">
        <span className="text-[hsl(var(--muted-foreground))]">{label}:</span>{' '}
        <span className="text-[hsl(var(--muted-foreground))]">N/A</span>
      </div>
    )
  }

  const formatted = formatJson(str)
  const lines = formatted.split('\n')
  const isLong = lines.length > 10 || str.length > 500

  const handleCopy = async () => {
    await navigator.clipboard.writeText(formatted)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="text-sm">
      <div className="flex items-center gap-2 mb-1">
        {isLong ? (
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex items-center gap-1 text-[hsl(var(--primary))] hover:underline"
          >
            {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRightIcon className="h-3 w-3" />}
            {label}
          </button>
        ) : (
          <span className="text-[hsl(var(--muted-foreground))]">{label}:</span>
        )}
        <button onClick={handleCopy} className="text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))]">
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
        </button>
      </div>
      {!isLong || expanded ? (
        <pre className="mt-1 max-h-64 overflow-auto rounded-md bg-[hsl(var(--muted))] p-2 text-xs font-mono whitespace-pre-wrap break-all">
          {formatted}
        </pre>
      ) : (
        <p className="text-xs text-[hsl(var(--muted-foreground))] truncate">{formatted.split('\n')[0]}...</p>
      )}
    </div>
  )
}

function ResponseHeaderDisplay({ data }: { data?: Record<string, string[]> }) {
  if (!data) {
    return (
      <div className="text-sm">
        <span className="text-[hsl(var(--muted-foreground))]">Response Headers:</span>{' '}
        <span className="text-[hsl(var(--muted-foreground))]">N/A</span>
      </div>
    )
  }

  return (
    <div className="text-sm">
      <span className="text-[hsl(var(--muted-foreground))]">Response Headers:</span>
      <div className="mt-1 max-h-32 overflow-auto rounded-md bg-[hsl(var(--muted))] p-2 text-xs font-mono">
        {Object.entries(data).map(([key, values]) => (
          <div key={key}>
            <span className="text-[hsl(var(--primary))]">{key}:</span>{' '}
            {Array.isArray(values) ? values.join(', ') : values}
          </div>
        ))}
      </div>
    </div>
  )
}

export function LogsPage() {
  const [logs, setLogs] = useState<RequestLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [modelFilter, setModelFilter] = useState<string>('')
  const [startTime, setStartTime] = useState<string>('')
  const [endTime, setEndTime] = useState<string>('')
  const [selectedLog, setSelectedLog] = useState<RequestLog | null>(null)
  const [providerCalls, setProviderCalls] = useState<ProviderCall[]>([])
  const [detailModalVisible, setDetailModalVisible] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      })

      if (statusFilter) params.append('status', statusFilter)
      if (modelFilter) params.append('model', modelFilter)
      if (startTime) params.append('start_time', new Date(startTime).toISOString())
      if (endTime) params.append('end_time', new Date(endTime).toISOString())

      const response = await fetch(`/api/logs/requests?${params}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('master_key')}`,
        },
      })

      if (response.ok) {
        const data = await response.json()
        setLogs(data.data || [])
        setTotal(data.total || 0)
      }
    } catch (error) {
      console.error('Failed to fetch logs:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchProviderCalls = async (requestLogId: number) => {
    try {
      const response = await fetch(`/api/logs/requests/${requestLogId}/calls`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('master_key')}`,
        },
      })

      if (response.ok) {
        const data = await response.json()
        setProviderCalls(data.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch provider calls:', error)
    }
  }

  const handleViewDetail = async (record: RequestLog) => {
    setSelectedLog(record)
    await fetchProviderCalls(record.id)
    setDetailModalVisible(true)
  }

  const handleDelete = async () => {
    try {
      const response = await fetch('/api/logs/requests?days=7', {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('master_key')}`,
        },
      })

      if (response.ok) {
        fetchLogs()
      }
    } catch (error) {
      console.error('Failed to delete logs:', error)
    } finally {
      setShowDeleteConfirm(false)
    }
  }

  useEffect(() => {
    fetchLogs()
  }, [page, pageSize, statusFilter, modelFilter, startTime, endTime])

  const totalPages = Math.ceil(total / pageSize)

  const formatTimestamp = (ts: string) => {
    const date = new Date(ts)
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Request Logs</h1>
          <p className="text-[hsl(var(--muted-foreground))]">View and manage API request logs</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={fetchLogs} disabled={loading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button variant="destructive" size="sm" onClick={() => setShowDeleteConfirm(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            Clean Old
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">All Requests</CardTitle>
          <CardDescription>{total} request(s) logged</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex flex-wrap items-center gap-3">
            <Input
              type="datetime-local"
              placeholder="Start time"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="w-52"
            />
            <Input
              type="datetime-local"
              placeholder="End time"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              className="w-52"
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="h-10 rounded-md border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2 text-sm"
            >
              <option value="">All Statuses</option>
              <option value="success">Success</option>
              <option value="failed">Failed</option>
            </select>
            <Input
              placeholder="Filter by model"
              value={modelFilter}
              onChange={(e) => setModelFilter(e.target.value)}
              className="w-48"
            />
          </div>

          {logs.length === 0 && !loading ? (
            <div className="flex flex-col items-center justify-center py-12">
              <FileText className="mb-4 h-12 w-12 text-[hsl(var(--muted-foreground))]" />
              <p className="text-lg font-medium">No logs found</p>
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                No request logs match your filters
              </p>
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Time</TableHead>
                    <TableHead>Request ID</TableHead>
                    <TableHead>Model</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Duration</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="font-mono text-sm">
                        {formatTimestamp(log.timestamp)}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        <span className="max-w-[160px] truncate block" title={log.request_id}>
                          {log.request_id}
                        </span>
                      </TableCell>
                      <TableCell>{log.model}</TableCell>
                      <TableCell>
                        <Badge variant={log.status === 'success' ? 'success' : 'destructive'}>
                          {log.status}
                        </Badge>
                      </TableCell>
                      <TableCell>{log.duration}ms</TableCell>
                      <TableCell>
                        <Button variant="ghost" size="sm" onClick={() => handleViewDetail(log)}>
                          <Eye className="mr-1 h-4 w-4" />
                          View
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              <div className="mt-4 flex items-center justify-between">
                <p className="text-sm text-[hsl(var(--muted-foreground))]">
                  Showing {logs.length} of {total} results
                </p>
                <div className="flex items-center gap-2">
                  <select
                    value={pageSize}
                    onChange={(e) => {
                      setPageSize(Number(e.target.value))
                      setPage(1)
                    }}
                    className="h-8 rounded-md border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-2 text-sm"
                  >
                    <option value={10}>10 / page</option>
                    <option value={20}>20 / page</option>
                    <option value={50}>50 / page</option>
                    <option value={100}>100 / page</option>
                  </select>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <span className="text-sm text-[hsl(var(--muted-foreground))]">
                    Page {page} of {totalPages || 1}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Detail Modal */}
      {detailModalVisible && selectedLog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="fixed inset-0 bg-black/50"
            onClick={() => setDetailModalVisible(false)}
          />
          <div className="relative z-50 w-full max-w-4xl max-h-[90vh] overflow-auto rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-6 shadow-lg">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-xl font-semibold">Request Details</h2>
              <Button variant="ghost" size="sm" onClick={() => setDetailModalVisible(false)}>
                <X className="h-4 w-4" />
              </Button>
            </div>

            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Request Info</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Request ID</dt>
                      <dd className="font-mono">{selectedLog.request_id}</dd>
                    </div>
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Model</dt>
                      <dd>{selectedLog.model}</dd>
                    </div>
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Method</dt>
                      <dd>{selectedLog.method}</dd>
                    </div>
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Path</dt>
                      <dd className="font-mono text-xs break-all">{selectedLog.path}</dd>
                    </div>
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Client IP</dt>
                      <dd className="font-mono">{selectedLog.client_ip}</dd>
                    </div>
                    <div>
                      <dt className="text-[hsl(var(--muted-foreground))]">Duration</dt>
                      <dd>{selectedLog.duration}ms</dd>
                    </div>
                  </dl>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Request Body</CardTitle>
                </CardHeader>
                <CardContent>
                  <CollapsibleJson label="Request Body" data={selectedLog.request_body} defaultExpanded={false} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Response Body</CardTitle>
                </CardHeader>
                <CardContent>
                  <CollapsibleJson label="Response Body" data={selectedLog.response_body} defaultExpanded={false} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Provider Calls</CardTitle>
                </CardHeader>
                <CardContent>
                  {providerCalls.length === 0 ? (
                    <p className="text-sm text-[hsl(var(--muted-foreground))]">
                      No provider calls recorded
                    </p>
                  ) : (
                    <div className="space-y-4">
                      {providerCalls.map((call) => (
                        <div key={call.id} className="rounded-md border border-[hsl(var(--border))] p-4">
                          <div className="mb-3 flex items-center gap-4 text-sm">
                            <span className="font-mono font-medium">{call.provider_id}</span>
                            <span>{call.model}</span>
                            <Badge
                              variant={
                                call.response_code >= 200 && call.response_code < 300
                                  ? 'success'
                                  : 'destructive'
                              }
                            >
                              {call.response_code}
                            </Badge>
                            <span>{call.duration}ms</span>
                            {call.is_retry && <Badge variant="warning">Retry</Badge>}
                            {call.error && <span className="text-destructive">{call.error}</span>}
                          </div>
                          <div className="space-y-2">
                            <CollapsibleJson label="Request Body" data={call.request_body} defaultExpanded={false} />
                            <ResponseHeaderDisplay data={call.response_header} />
                            <CollapsibleJson label="Response Body" data={call.response_body} defaultExpanded={false} />
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowDeleteConfirm(false)} />
          <div className="relative z-50 w-full max-w-md rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-6 shadow-lg">
            <h2 className="mb-2 text-lg font-semibold">Confirm Delete</h2>
            <p className="mb-6 text-sm text-[hsl(var(--muted-foreground))]">
              Are you sure you want to delete logs older than 7 days? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setShowDeleteConfirm(false)}>
                Cancel
              </Button>
              <Button variant="destructive" size="sm" onClick={handleDelete}>
                Delete
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default LogsPage
