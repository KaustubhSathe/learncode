"use client"

import { createContext, useContext, useEffect, useState } from "react"
import { useRouter, usePathname } from "next/navigation"

interface User {
  id: number
  login: string
  email: string
}

interface AuthContextType {
  user: User | null
  loading: boolean
  logout: () => void
  authToken: string | null
  checkAuth: () => Promise<void>
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  logout: () => {},
  authToken: null,
  checkAuth: async () => {},
})

function AuthContent({ children }: { children: React.ReactNode }) {
  const { loading } = useAuth()

  if (loading) {
    return (
      <div suppressHydrationWarning className="min-h-screen bg-background">
        <div className="container mx-auto py-6">
          <div className="flex items-center justify-center min-h-[calc(100vh-4rem)]">
            Loading...
          </div>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [authToken, setAuthToken] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    checkAuth()
  }, [])

  const checkAuth = async () => {
    const token = localStorage.getItem('auth_token')
    if (!token) {
      setLoading(false)
      setAuthToken(null)
      if (pathname !== '/') {
        router.push('/')
      }
      return
    }

    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/auth/verify`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error('Invalid token')
      }

      const data = await response.json()
      setUser(data)
      setAuthToken(token)
      setLoading(false)
    } catch (error) {
      localStorage.removeItem('auth_token')
      if (pathname !== '/') {
        router.push('/')
      }
      setAuthToken(null)
      setLoading(false)
    }
  }

  const logout = () => {
    localStorage.removeItem('auth_token')
    setUser(null)
    setAuthToken(null)
    router.push('/')
  }

  return (
    <AuthContext.Provider value={{ user, loading, logout, authToken, checkAuth }}>
      <AuthContent>{children}</AuthContent>
    </AuthContext.Provider>
  )
}

export const useAuth = () => useContext(AuthContext) 