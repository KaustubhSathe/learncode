"use client"

import Link from "next/link"
import { ModeToggle } from "./mode-toggle"
import { LogOut } from "lucide-react"
import { usePathname } from "next/navigation"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/lib/auth"

export default function Navbar() {
  const pathname = usePathname()
  const { authToken, logout } = useAuth()

  if (!pathname.startsWith('/problems')) {
    return null
  }

  return (
    <nav className="border-b">
      <div className="container mx-auto flex h-16 items-center px-4">
        <Link 
          href={authToken ? "/problems" : "/"} 
          className="text-xl font-bold"
        >
          LearnCode
        </Link>
        
        <div className="flex items-center space-x-4 ml-auto">
          {authToken && (
            <Button 
              variant="destructive"
              onClick={logout}
              size="sm"
              className="flex items-center gap-2"
            >
              <LogOut className="h-4 w-4" />
              Logout
            </Button>
          )}
          <ModeToggle />
        </div>
      </div>
    </nav>
  )
} 