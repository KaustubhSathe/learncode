'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Problem } from '@/types'
import Link from 'next/link'
import { useAuth } from '@/lib/auth'

export default function ProblemsPage() {
  const router = useRouter()
  const [problems, setProblems] = useState<Problem[]>([])
  const { user } = useAuth()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchProblems = async () => {
      try {
        const token = localStorage.getItem('auth_token')
        
        if (!token) {
          router.push('/')
          return
        }

        const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/problems`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        })

        if (!response.ok) {
          const errorText = await response.text()
          throw new Error('Failed to fetch problems')
        }

        const data = await response.json()
        
        setProblems(data.problems)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred')
      } finally {
        setLoading(false)
      }
    }

    fetchProblems()
  }, [router])


  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          {user && <h2 className="text-xl font-bold">Welcome, {user.login}!</h2>}
        </div>
      </div>
      
      <h1 className="text-3xl font-bold mb-6">Coding Problems</h1>
      
      <div className="grid gap-4">
        {problems.map((problem) => (
          <div 
            key={problem.id} 
            className="border rounded-lg p-4 hover:shadow-lg transition-shadow cursor-pointer"
            onClick={() => {
              router.push(`/problems/${problem.id}`)
            }}
          >
            <div className="flex justify-between items-center">
              <h3 className="text-xl font-semibold">{problem.title}</h3>
              <span className={`px-3 py-1 rounded-full text-sm ${
                problem.difficulty === 'Easy' ? 'bg-green-100 text-green-800' :
                problem.difficulty === 'Medium' ? 'bg-yellow-100 text-yellow-800' :
                'bg-red-100 text-red-800'
              }`}>
                {problem.difficulty}
              </span>
            </div>
            <p className="mt-2 text-gray-600">{problem.description}</p>
          </div>
        ))}
      </div>

      {problems.length === 0 && (
        <div className="text-center text-gray-500 mt-8">
          No problems found
        </div>
      )}
    </div>
  )
}