'use client'

import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

export default function AuthCallback() {
  const router = useRouter()
  const searchParams = useSearchParams()

  useEffect(() => {
    const token = searchParams.get('token')
    console.log('Callback: Received token from URL:', token ? 'yes' : 'no')

    if (token) {
      try {
        // Store the token
        localStorage.setItem('auth_token', token)
        // Verify token was stored
        const storedToken = localStorage.getItem('auth_token')
        console.log('Callback: Token stored successfully:', storedToken ? 'yes' : 'no')
        
        // Small delay to ensure token is stored
        setTimeout(() => {
          router.push('/problems')
        }, 100)
      } catch (error) {
        router.push('/')
      }
    } else {
      router.push('/')
    }
  }, [searchParams, router])

  return <div>Loading...</div>
} 