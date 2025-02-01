'use client'

import { useEffect } from 'react'
import { useRouter, usePathname } from 'next/navigation'

export default function AuthCheck({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    const publicPaths = ['/', '/login', '/auth/callback']
    
    if (!token && !publicPaths.some(path => pathname.startsWith(path))) {
      router.push('/login')
    }
  }, [pathname, router])

  return <>{children}</>
} 