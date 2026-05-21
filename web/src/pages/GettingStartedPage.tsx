import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Copy, Check, BookOpen, Terminal, Code2 } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/Button'

function CodeBlock({ code, language }: { code: string; language?: string }) {
  const [copied, setCopied] = useState(false)

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="relative mt-2 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--muted))]">
      <div className="flex items-center justify-between px-4 py-1.5 border-b border-[hsl(var(--border))] bg-[hsl(var(--card))] rounded-t-lg">
        <span className="text-xs text-[hsl(var(--muted-foreground))]">{language || 'plain'}</span>
        <Button variant="ghost" size="sm" onClick={copyToClipboard}>
          {copied ? (
            <Check className="h-3.5 w-3.5 text-[hsl(var(--success))]" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </Button>
      </div>
      <pre className="overflow-x-auto p-4 text-sm leading-relaxed">
        <code>{code}</code>
      </pre>
    </div>
  )
}

export function GettingStartedPage() {
  const host = window.location.host
  const baseURL = `${window.location.protocol}//${host}`

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Getting Started</h1>
        <p className="mt-2 text-[hsl(var(--muted-foreground))]">
          Connect your LLM clients to LLMux using either OpenAI or Anthropic API format.
          Protocol conversion happens automatically — you can access any provider through either interface.
        </p>
      </div>

      {/* OpenAI Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Terminal className="h-5 w-5 text-[hsl(var(--primary))]" />
            <CardTitle className="text-lg">OpenAI-compatible API</CardTitle>
          </div>
          <CardDescription>
            Use any OpenAI SDK or client (chatgpt-cli, langchain, copilot, etc.) by pointing to LLMux.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h4 className="text-sm font-medium">Base URL</h4>
            <CodeBlock code={`${baseURL}/v1`} language="url" />
          </div>

          <div>
            <h4 className="text-sm font-medium">Authentication</h4>
            <CodeBlock
              language="header"
              code={`Authorization: Bearer <your-api-key>`}
            />
          </div>

          <div>
            <h4 className="text-sm font-medium">Example: cURL</h4>
            <CodeBlock
              language="bash"
              code={`curl ${baseURL}/v1/chat/completions \\
  -H "Authorization: Bearer <your-api-key>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "<alias-or-provider/model>",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'`}
            />
          </div>

          <div>
            <h4 className="text-sm font-medium">Example: OpenAI Python SDK</h4>
            <CodeBlock
              language="python"
              code={`from openai import OpenAI

client = OpenAI(
    base_url="${baseURL}/v1",
    api_key="<your-api-key>",
)

response = client.chat.completions.create(
    model="<alias-or-provider/model>",
    messages=[{"role": "user", "content": "Hello!"}],
    stream=True,
)
for chunk in response:
    print(chunk.choices[0].delta.content, end="")`}
            />
          </div>

          <div>
            <h4 className="text-sm font-medium">Available Endpoints</h4>
            <CodeBlock
              language="plain"
              code={`POST ${baseURL}/v1/chat/completions    # Chat completions (streaming supported)
GET  ${baseURL}/v1/models               # List available models
POST ${baseURL}/v1/v1/chat/completions   # Compatible path`}
            />
          </div>
        </CardContent>
      </Card>

      {/* Anthropic Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Code2 className="h-5 w-5 text-[hsl(var(--primary))]" />
            <CardTitle className="text-lg">Anthropic-compatible API</CardTitle>
          </div>
          <CardDescription>
            Use the Anthropic SDK or Messages API format. LLMux auto-converts between OpenAI and Anthropic protocols.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h4 className="text-sm font-medium">Base URL</h4>
            <CodeBlock code={`${baseURL}/anthropic`} language="url" />
          </div>

          <div>
            <h4 className="text-sm font-medium">Authentication</h4>
            <CodeBlock
              language="header"
              code={`Authorization: Bearer <your-api-key>
anthropic-version: 2023-06-01  # optional, auto-added if missing`}
            />
          </div>

          <div>
            <h4 className="text-sm font-medium">Example: Messages API</h4>
            <CodeBlock
              language="bash"
              code={`curl ${baseURL}/anthropic/v1/messages \\
  -H "Authorization: Bearer <your-api-key>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "<alias-or-provider/model>",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 4096,
    "stream": true
  }'`}
            />
          </div>

          <div>
            <h4 className="text-sm font-medium">Available Endpoints</h4>
            <CodeBlock
              language="plain"
              code={`POST ${baseURL}/anthropic/v1/messages    # Messages (streaming supported)`}
            />
          </div>
        </CardContent>
      </Card>

      {/* Model format */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <BookOpen className="h-5 w-5 text-[hsl(var(--primary))]" />
            <CardTitle className="text-lg">Model Format</CardTitle>
          </div>
          <CardDescription>
            Two ways to specify which model to use.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <h4 className="text-sm font-medium">Alias (recommended)</h4>
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Use the alias name configured in the Aliases page. Supports round-robin, random, and fallback strategies across multiple providers.
            </p>
            <CodeBlock code={`{"model": "my-alias-name"}`} language="json" />
          </div>
          <div>
            <h4 className="text-sm font-medium">Direct reference</h4>
            <p className="text-sm text-[hsl(var(--muted-foreground))]">
              Reference a specific provider and model directly using <code>provider_id/model_name</code> format.
            </p>
            <CodeBlock code={`{"model": "deepseek/deepseek-v4-flash"}`} language="json" />
          </div>
        </CardContent>
      </Card>

      {/* Protocol Conversion */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Protocol Conversion</CardTitle>
          <CardDescription>
            LLMux automatically converts requests and responses between OpenAI and Anthropic formats, including SSE streaming.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[hsl(var(--border))]">
                  <th className="text-left py-2 pr-4 font-medium">Client Protocol</th>
                  <th className="text-left py-2 pr-4 font-medium">Provider Type</th>
                  <th className="text-left py-2 font-medium">Result</th>
                </tr>
              </thead>
              <tbody className="text-[hsl(var(--muted-foreground))]">
                <tr className="border-b border-[hsl(var(--border))]">
                  <td className="py-2 pr-4">OpenAI</td>
                  <td className="py-2 pr-4">OpenAI</td>
                  <td className="py-2">Passthrough (no conversion)</td>
                </tr>
                <tr className="border-b border-[hsl(var(--border))]">
                  <td className="py-2 pr-4">OpenAI</td>
                  <td className="py-2 pr-4">Anthropic</td>
                  <td className="py-2">Auto-converted</td>
                </tr>
                <tr className="border-b border-[hsl(var(--border))]">
                  <td className="py-2 pr-4">Anthropic</td>
                  <td className="py-2 pr-4">OpenAI</td>
                  <td className="py-2">Auto-converted</td>
                </tr>
                <tr>
                  <td className="py-2 pr-4">Anthropic</td>
                  <td className="py-2 pr-4">Anthropic</td>
                  <td className="py-2">Passthrough (no conversion)</td>
                </tr>
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
