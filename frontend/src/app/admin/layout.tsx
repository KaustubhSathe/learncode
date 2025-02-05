'use client'

import { useRouter } from "next/navigation"
import { useEffect } from "react"
import Navbar from "@/components/navbar"
import CheckAuth from "@/components/check-auth"
import { useAppSelector } from "@/store/hooks"
import { User } from "@/types"

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <CheckAuth>
        <div className="min-h-screen bg-background">
            <Navbar />
            {children}
        </div>
    </CheckAuth>
  )
} 