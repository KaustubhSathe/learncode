"use client"

import { useEffect } from "react"
import { Github } from "lucide-react"
import Cookies from 'js-cookie'
import { useRouter } from "next/navigation"

export default function LoginPage() {
  const router = useRouter()

  // Handle GitHub OAuth callback
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get("token")
    if (token) {
      // Store token in cookie
      Cookies.set('auth_token', token, { 
        expires: 7, // 7 days
        secure: true,
        sameSite: 'strict'
      })
      // Redirect to home
      router.push('/')
    }
  }, [router])

  const handleGitHubLogin = () => {
    // Redirect to backend auth endpoint
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/auth/github`
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-4rem)]">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold">Welcome to LearnCode</h2>
          <p className="mt-2 text-muted-foreground">
            Sign in to start solving problems
          </p>
        </div>

        <button
          onClick={handleGitHubLogin}
          className="flex items-center justify-center w-full gap-2 px-4 py-2 text-white bg-[#24292F] hover:bg-[#24292F]/90 rounded-md"
        >
          <Github className="w-5 h-5" />
          Sign in with GitHub
        </button>
      </div>
    </div>
  )
} 