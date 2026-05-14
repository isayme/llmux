import type { ProvidersResponse, APIKeysResponse, AliasesResponse, VersionInfo } from '@/types'

const API_BASE = ''

class ApiService {
  private getHeaders(): HeadersInit {
    return {
      'Content-Type': 'application/json',
    }
  }

  async getVersion(): Promise<VersionInfo> {
    const response = await fetch(`${API_BASE}/version`)
    if (!response.ok) {
      throw new Error('Failed to fetch version')
    }
    return response.json()
  }

  async login(masterKey: string): Promise<boolean> {
    const response = await fetch(`${API_BASE}/api/login`, {
      method: 'POST',
      headers: this.getHeaders(),
      credentials: 'include',
      body: JSON.stringify({ master_key: masterKey }),
    })
    return response.ok
  }

  async logout(): Promise<void> {
    await fetch(`${API_BASE}/api/logout`, {
      method: 'POST',
      headers: this.getHeaders(),
      credentials: 'include',
    })
  }

  async checkSession(): Promise<boolean> {
    const response = await fetch(`${API_BASE}/api/check`, {
      headers: this.getHeaders(),
      credentials: 'include',
    })
    return response.ok
  }

  async getProviders(): Promise<ProvidersResponse> {
    const response = await fetch(`${API_BASE}/api/providers`, {
      headers: this.getHeaders(),
      credentials: 'include',
    })
    if (response.status === 401) {
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
      credentials: 'include',
    })
    if (response.status === 401) {
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
      credentials: 'include',
    })
    if (response.status === 401) {
      throw new Error('Unauthorized')
    }
    if (!response.ok) {
      throw new Error('Failed to fetch aliases')
    }
    return response.json()
  }
}

export const api = new ApiService()
