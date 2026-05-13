const API_BASE = '/api'

export interface ProviderConfig {
  id: string
  name: string
  api_key: string
  base_url: string
  type: string
  enabled: boolean
}

export interface ApiKeyConfig {
  name: string
  key: string
  enabled: boolean
}

export interface ModelAlias {
  name: string
  provider: string
  model: string
  enabled: boolean
}

export interface AliasesResponse {
  aliases: Record<string, ModelAlias>
}

class ApiService {
  private getMasterKey(): string {
    return localStorage.getItem('masterKey') || ''
  }

  private getHeaders(): HeadersInit {
    return {
      'Authorization': `Bearer ${this.getMasterKey()}`,
      'Content-Type': 'application/json',
    }
  }

  async login(masterKey: string): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE}/providers`, {
        headers: {
          'Authorization': `Bearer ${masterKey}`,
        },
      })
      if (response.ok) {
        localStorage.setItem('masterKey', masterKey)
        return true
      }
      return false
    } catch {
      return false
    }
  }

  checkAuth(): boolean {
    const key = this.getMasterKey()
    return !!key
  }

  logout() {
    localStorage.removeItem('masterKey')
  }

  async getProviders(): Promise<ProviderConfig[]> {
    const response = await fetch(`${API_BASE}/providers`, {
      headers: this.getHeaders(),
    })
    if (!response.ok) {
      throw new Error('Failed to fetch providers')
    }
    const data = await response.json()
    return data.providers || []
  }

  async getApiKeys(): Promise<ApiKeyConfig[]> {
    const response = await fetch(`${API_BASE}/api-keys`, {
      headers: this.getHeaders(),
    })
    if (!response.ok) {
      throw new Error('Failed to fetch api keys')
    }
    const data = await response.json()
    return data.api_keys || []
  }

  async getAliases(): Promise<AliasesResponse> {
    const response = await fetch(`${API_BASE}/aliases`, {
      method: 'POST',
      headers: this.getHeaders(),
    })
    if (!response.ok) {
      throw new Error('Failed to fetch aliases')
    }
    return response.json()
  }
}

export const apiService = new ApiService()