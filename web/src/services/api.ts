import type { ProvidersResponse, APIKeysResponse, AliasesResponse, VersionInfo } from '@/types'

const API_BASE = ''

class ApiService {
  private masterKey: string | null = null

  setMasterKey(key: string) {
    this.masterKey = key
    localStorage.setItem('llmux_master_key', key)
  }

  getMasterKey(): string | null {
    if (!this.masterKey) {
      this.masterKey = localStorage.getItem('llmux_master_key')
    }
    return this.masterKey
  }

  clearMasterKey() {
    this.masterKey = null
    localStorage.removeItem('llmux_master_key')
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }
    if (this.masterKey) {
      headers['Authorization'] = `Bearer ${this.masterKey}`
    }
    return headers
  }

  async getVersion(): Promise<VersionInfo> {
    const response = await fetch(`${API_BASE}/version`)
    if (!response.ok) {
      throw new Error('Failed to fetch version')
    }
    return response.json()
  }

  async getProviders(): Promise<ProvidersResponse> {
    const response = await fetch(`${API_BASE}/api/providers`, {
      headers: this.getHeaders(),
    })
    if (response.status === 401) {
      this.clearMasterKey()
      throw new Error('Unauthorized')
    }
    if (!response.ok) {
      throw new Error('Failed to fetch providers')
    }
    return response.json()
  }

  async getAPIKeys(): Promise<APIKeysResponse> {
    const response = await fetch(`${API_BASE}/api/api-keys`, {
      headers: this.getHeaders(),
    })
    if (response.status === 401) {
      this.clearMasterKey()
      throw new Error('Unauthorized')
    }
    if (!response.ok) {
      throw new Error('Failed to fetch API keys')
    }
    return response.json()
  }

  async getAliases(): Promise<AliasesResponse> {
    const response = await fetch(`${API_BASE}/api/aliases`, {
      method: 'POST',
      headers: this.getHeaders(),
    })
    if (response.status === 401) {
      this.clearMasterKey()
      throw new Error('Unauthorized')
    }
    if (!response.ok) {
      throw new Error('Failed to fetch aliases')
    }
    return response.json()
  }

  async validateMasterKey(key: string): Promise<boolean> {
    const tempKey = this.masterKey
    this.masterKey = key
    try {
      await this.getProviders()
      return true
    } catch {
      this.masterKey = tempKey
      return false
    }
  }
}

export const api = new ApiService()
