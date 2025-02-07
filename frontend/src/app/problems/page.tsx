'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Problem } from '@/types'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import Link from 'next/link'

export default function ProblemsPage() {
  const router = useRouter()
  const [problems, setProblems] = useState<Problem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>('all')

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
          throw new Error('Failed to fetch problems', { cause: errorText })
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

  const filteredProblems = problems.filter(problem => 
    selectedDifficulty === 'all' || problem.difficulty === selectedDifficulty
  )

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>

  return (
    <div className="container mx-auto py-8 px-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Problems</h1>
        <Select value={selectedDifficulty} onValueChange={setSelectedDifficulty}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter by difficulty" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Difficulties</SelectItem>
            <SelectItem value="Easy">Easy</SelectItem>
            <SelectItem value="Medium">Medium</SelectItem>
            <SelectItem value="Hard">Hard</SelectItem>
          </SelectContent>
        </Select>
      </div>
      
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Title</TableHead>
            <TableHead>Difficulty</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filteredProblems.map((problem) => (
            <TableRow 
              key={problem.id}
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => router.push(`/problems/${problem.id}`)}
            >
              <TableCell className="font-medium">
                {problem.title}
              </TableCell>
              <TableCell>
                <span className={
                  problem.difficulty === 'Easy' ? 'text-green-500' :
                  problem.difficulty === 'Medium' ? 'text-yellow-500' :
                  'text-red-500'
                }>
                  {problem.difficulty}
                </span>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {problems.length === 0 && (
        <div className="text-center text-gray-500 dark:text-gray-400 mt-8">
          No problems found
        </div>
      )}
    </div>
  )
}