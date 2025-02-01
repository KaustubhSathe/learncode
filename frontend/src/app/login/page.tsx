"use client"

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

export default function LoginPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    console.log('Login: Token retrieved:', token)
    if (token) {
      router.push('/problems')
    }
    setIsLoading(false)
  }, [router])

  const handleGitHubLogin = () => {
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/auth/github`
  }

  if (isLoading) {
    return <div>Loading...</div>
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <button
        onClick={handleGitHubLogin}
        className="bg-gray-900 text-white px-6 py-3 rounded-lg flex items-center gap-2 hover:bg-gray-800"
      >
        <span>Login with GitHub</span>
      </button>
    </div>
  )
} 