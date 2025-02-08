'use client'

import { Suspense } from 'react'
import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

function CallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()

  useEffect(() => {
    const token = searchParams.get('token')

    if (token) {
      try {
        // Store the token
        localStorage.setItem('auth_token', token)
        // Verify token was stored
        const storedToken = localStorage.getItem('auth_token')
        // Trigger auth check to update context
        router.push('/problems')
      } catch (error) {
        router.push('/')
      }
    } else {
      router.push('/')
    }
  }, [searchParams, router])

  return <div>Loading...</div>
}

export default function AuthCallback() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <CallbackContent />
    </Suspense>
  )
} 