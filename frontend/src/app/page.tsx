'use client'

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"

export default function Home() {
  const router = useRouter()

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      router.push('/problems')
    }
  }, [router])

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-4rem)] text-center">
      <h1 className="text-4xl font-bold tracking-tight lg:text-5xl">
        Practice Programming Problems
      </h1>
      <p className="mt-4 text-lg text-muted-foreground">
        Improve your coding skills by solving programming challenges
      </p>
      <div className="mt-8">
        <Link 
          href="/login"
          className="rounded-md bg-primary px-4 py-2 text-primary-foreground hover:bg-primary/90"
        >
          Start Practicing
        </Link>
      </div>
    </div>
  )
} 