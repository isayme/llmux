import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { api } from '@/services/api'

interface AuthContextType {
  isAuthenticated: boolean
  login: (masterKey: string) => Promise<boolean>
  logout: () => Promise<void>
  isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const isValid = await api.checkSession()
        setIsAuthenticated(isValid)
      } catch {
        setIsAuthenticated(false)
      }
      setIsLoading(false)
    }
    checkAuth()
  }, [])

  const login = async (masterKey: string): Promise<boolean> => {
    const success = await api.login(masterKey)
    if (success) {
      setIsAuthenticated(true)
      return true
    }
    return false
  }

  const logout = async () => {
    await api.logout()
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
