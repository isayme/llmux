import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { useAuth } from '@/contexts/AuthContext'
import { AlertCircle, Key } from 'lucide-react'
import { useState } from 'react'

export function LoginPage() {
  const { login } = useAuth()
  const [masterKey, setMasterKey] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      const success = await login(masterKey)
      if (!success) {
        setError('Invalid Master Key')
      }
    } catch {
      setError('Failed to connect to server')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[hsl(var(--background))] p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-[hsl(var(--primary))]">
            <Key className="h-6 w-6 text-[hsl(var(--primary-foreground))]" />
          </div>
          <CardTitle>LLMux Admin</CardTitle>
          <CardDescription>Enter your Master Key to access the admin panel</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="masterKey" className="block text-sm font-medium mb-4">
                Master Key
              </label>
              <Input
                id="masterKey"
                type="password"
                placeholder="Enter your master key"
                value={masterKey}
                onChange={(e) => setMasterKey(e.target.value)}
                required
              />
            </div>

            {error && (
              <div className="flex items-center gap-2 text-sm text-[hsl(var(--destructive))]">
                <AlertCircle className="h-4 w-4" />
                {error}
              </div>
            )}

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? 'Authenticating...' : 'Login'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
