"use client"

import { useAppSelector } from "@/store/hooks"
import Navbar from "@/components/navbar"
import CheckAuth from "@/components/check-auth"

export default function ProblemsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { loading } = useAppSelector(state => state.auth)

  if (loading) {
    return <div>Loading...</div>
  }

  return (
    <CheckAuth>
      <div className="h-screen bg-background flex flex-col">
        <Navbar />
        <main className="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-2 
          [&::-webkit-scrollbar-track]:bg-transparent
          [&::-webkit-scrollbar-thumb]:bg-gray-300 
          [&::-webkit-scrollbar-thumb:hover]:bg-gray-400
          dark:[&::-webkit-scrollbar-thumb]:bg-gray-700
          dark:[&::-webkit-scrollbar-thumb:hover]:bg-gray-600
          [&::-webkit-scrollbar-thumb]:rounded-full"
        >
          {children}
        </main>
      </div>
    </CheckAuth>
  )
} 