"use client"

import { AuthProvider, useAuth } from "@/lib/auth"
import Navbar from "@/components/navbar"
import { useRouter } from "next/navigation"
import { useEffect } from "react"

export default function ProblemsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthProvider>
            <div className="min-h-screen bg-background">
      <Navbar />
      <main className="container mx-auto py-6">
        {children}
      </main>
    </div>
    </AuthProvider>
  )
} 