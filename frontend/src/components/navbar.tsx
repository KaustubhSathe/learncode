"use client"

import Link from "next/link"
import { ModeToggle } from "./mode-toggle"
import { useAuth } from "@/lib/auth"
import { LogOut, User } from "lucide-react"

export default function Navbar() {
  const { user, logout } = useAuth()

  return (
    <nav className="border-b">
      <div className="container mx-auto flex h-16 items-center px-4">
        <Link href="/" className="text-xl font-bold">
          LearnCode
        </Link>
        
        <div className="flex items-center space-x-4 ml-auto">
          <Link href="/problems" className="hover:text-primary">
            Problems
          </Link>
          {user ? (
            <>
              <div className="flex items-center space-x-2">
                <User className="w-4 h-4" />
                <span>{user.login}</span>
              </div>
              <button onClick={logout} className="hover:text-primary">
                <LogOut className="w-4 h-4" />
              </button>
            </>
          ) : (
            <Link href="/login" className="hover:text-primary">
              Login
            </Link>
          )}
          <ModeToggle />
        </div>
      </div>
    </nav>
  )
} 