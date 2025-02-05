'use client'

import { Inter } from "next/font/google"
import "./globals.css"
import { ThemeProvider } from "@/components/theme-provider"
import { Provider } from 'react-redux'
import { store } from '@/store'
import CheckAuth from '@/components/check-auth'

const inter = Inter({ subsets: ["latin"] })

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <Provider store={store}>
          <ThemeProvider
            attribute="class"
            defaultTheme="dark"
            enableSystem
            disableTransitionOnChange
          >
            <CheckAuth>
              {children}
            </CheckAuth>
          </ThemeProvider>
        </Provider>
      </body>
    </html>
  )
} 