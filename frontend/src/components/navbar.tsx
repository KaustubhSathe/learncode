"use client"

import Link from "next/link"
import { ModeToggle } from "./mode-toggle"
import { LogOut } from "lucide-react"
import { usePathname } from "next/navigation"
import { Button } from "@/components/ui/button"
import { useAppSelector, useAppDispatch } from "@/store/hooks"
import { logout } from "@/store/auth-slice"
import { useRouter } from "next/navigation"

export default function Navbar() {
  const pathname = usePathname()
  const { user } = useAppSelector(state => state.auth)
  const dispatch = useAppDispatch()
  const router = useRouter()
  if (!pathname.startsWith('/problems')) {
    return null
  }


  return (
    <nav className="border-b">
      <div className="container mx-auto flex h-16 items-center px-4">
        <Link 
          href={user ? "/problems" : "/"} 
          className="text-xl font-bold"
        >
          LearnCode
        </Link>
        
        <div className="flex items-center space-x-4 ml-auto">
           
            <Button 
              variant="destructive"
              onClick={() => {
                dispatch(logout())
                router.push("/")
              }}
              size="sm"
              className="flex items-center gap-2"
            >
              <LogOut className="h-4 w-4" />
              Logout
            </Button>
          
          <ModeToggle />
        </div>
      </div>
    </nav>
  )
} 