'use client'

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { Github } from "lucide-react"
import { Button } from "@/components/ui/button"

export default function Home() {
  const router = useRouter()
  
  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      router.push('/problems')
    }
  }, [router])

  const handleGitHubLogin = () => {
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/auth/github`
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-4rem)] text-center">
      <h1 className="text-4xl font-bold tracking-tight lg:text-5xl">
        Practice Programming Problems
      </h1>
      <p className="mt-4 text-lg text-muted-foreground">
        Improve your coding skills by solving programming challenges
      </p>
      <div className="mt-8">
        <Button
          onClick={handleGitHubLogin}
          size="lg"
          className="flex items-center gap-2"
        >
          <Github className="w-5 h-5" />
          Login with GitHub
        </Button>
      </div>
    </div>
  )
} 