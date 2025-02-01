'use client'

import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

export default function AuthCallback() {
  const router = useRouter()
  const searchParams = useSearchParams()

  useEffect(() => {
    const token = searchParams.get('token')
    console.log('Callback: Token received:', token)

    if (token) {
      try {
        // Store the token
        localStorage.setItem('auth_token', token)
        // Verify token was stored
        const storedToken = localStorage.getItem('auth_token')
        console.log('Callback: Token stored:', storedToken)
        
        // Small delay to ensure token is stored
        setTimeout(() => {
          router.push('/problems')
        }, 100)
      } catch (error) {
        console.error('Error storing token:', error)
        router.push('/')
      }
    } else {
      console.log('No token found in URL')
      router.push('/')
    }
  }, [searchParams, router])

  return <div>Loading...</div>
} 