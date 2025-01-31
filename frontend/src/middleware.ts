import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Get token from cookies
  const token = request.cookies.get('auth_token')?.value

  // Get the pathname
  const { pathname } = request.nextUrl

  // Allow access to login page and auth callback
  if (pathname.startsWith('/login') || pathname.startsWith('/auth/callback')) {
    return NextResponse.next()
  }

  // Redirect to login if no token is present
  if (!token) {
    const loginUrl = new URL('/login', request.url)
    return NextResponse.redirect(loginUrl)
  }

  return NextResponse.next()
}

// Configure which paths should be protected
export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - login page
     */
    '/((?!_next/static|_next/image|favicon.ico|login).*)',
  ],
} 