import Link from "next/link"
import { ModeToggle } from "./mode-toggle"

export default function Navbar() {
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
          <Link href="/submissions" className="hover:text-primary">
            Submissions
          </Link>
          <ModeToggle />
        </div>
      </div>
    </nav>
  )
} 