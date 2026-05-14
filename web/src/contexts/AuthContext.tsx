import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { api } from '@/services/api'

interface AuthContextType {
  isAuthenticated: boolean
  login: (masterKey: string) => Promise<boolean>
  logout: () => void
  isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const checkAuth = async () => {
      const savedKey = api.getMasterKey()
      if (savedKey) {
        try {
          await api.getProviders()
          setIsAuthenticated(true)
        } catch {
          api.clearMasterKey()
          setIsAuthenticated(false)
        }
      }
      setIsLoading(false)
    }
    checkAuth()
  }, [])

  const login = async (masterKey: string): Promise<boolean> => {
    const isValid = await api.validateMasterKey(masterKey)
    if (isValid) {
      api.setMasterKey(masterKey)
      setIsAuthenticated(true)
      return true
    }
    return false
  }

  const logout = () => {
    api.clearMasterKey()
    setIsAuthenticated(false)
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
