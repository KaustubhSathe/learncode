'use client'

import { useEffect } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import { useAppDispatch } from '@/store/hooks'
import { setUser, setLoading } from '@/store/auth-slice'

export default function CheckAuth({ children }: { children: React.ReactNode }) {
  const dispatch = useAppDispatch()
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('auth_token')
      
      if (!token) {
        dispatch(setLoading(true))
        dispatch(setUser(null))
        if (pathname !== '/') {
          router.push('/')
        }
        return
      }

      try {
        const response = await fetch(`${process.env.API_URL}/auth/verify`, {
          headers: {
            authorization: `Bearer ${token}`,
          },
        })

        if (!response.ok) {
          throw new Error('Invalid token', { cause: response.statusText })
        }

        const data = await response.json()
        dispatch(setUser(data))
        dispatch(setLoading(false))
      } catch (error) {
        localStorage.removeItem('auth_token')
        dispatch(setUser(null))
        dispatch(setLoading(true))
        if (pathname !== '/') {
          router.push('/')
        }
      }
    }

    checkAuth()
  }, [dispatch, router, pathname])

  return <>{children}</>
} 