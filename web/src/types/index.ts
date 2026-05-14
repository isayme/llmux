export interface Provider {
  id: string
  name: string
  type: 'openai' | 'anthropic'
  base_url: string
  api_key?: string
  enabled: boolean
}

export interface APIKey {
  name: string
  key: string
  enabled: boolean
}

export interface Alias {
  name: string
  provider: string
  model: string
  enabled: boolean
}

export interface VersionInfo {
  version: string
}

export interface ProvidersResponse {
  providers: Provider[]
}

export interface APIKeysResponse {
  api_keys: APIKey[]
}

export interface AliasesResponse {
  aliases: Record<string, Alias>
}
