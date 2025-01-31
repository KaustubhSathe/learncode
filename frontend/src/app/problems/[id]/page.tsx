"use client"

import { useState, useEffect } from "react"
import Editor from "@monaco-editor/react"
import { ChevronDown, ChevronUp, Play } from "lucide-react"
import { submitCode, getSubmissionStatus } from '@/lib/submission'
import { supabase } from '@/lib/supabase'

const defaultCode = `function solution(nums, target) {
  // Write your code here
}`

export default function ProblemPage({ params }: { params: { id: string } }) {
  const [code, setCode] = useState(defaultCode)
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(true)
  const [output, setOutput] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [testInput, setTestInput] = useState("")
  const [expectedOutput, setExpectedOutput] = useState("")

  useEffect(() => {
    // Fetch problem details including input and output
    const fetchProblem = async () => {
      const { data: problem } = await supabase
        .from('problems')
        .select('*')
        .is('deleted_at', null)
        .eq('id', params.id)
        .single()

      if (problem) {
        setTestInput(problem.input)
        setExpectedOutput(problem.output)
      }
    }

    fetchProblem()
  }, [params.id])

  const handleSubmit = async () => {
    try {
      setIsSubmitting(true)
      const submission = await submitCode({
        problemId: parseInt(params.id),
        code,
        language: 'javascript', // or get from state if you have language selection
      })

      // Poll for submission status
      const interval = setInterval(async () => {
        const status = await getSubmissionStatus(submission.id)
        if (status.completed_at) {
          clearInterval(interval)
          setOutput(JSON.stringify(status.results, null, 2))
        }
      }, 1000)

    } catch (error) {
      console.error('Error submitting code:', error)
      setOutput('Error submitting code')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="h-[calc(100vh-4rem)] flex flex-col">
      {/* Problem Description */}
      <div className="border-b">
        <button
          onClick={() => setIsDescriptionExpanded(!isDescriptionExpanded)}
          className="flex items-center justify-between w-full p-4 text-left"
        >
          <h1 className="text-xl font-bold">Two Sum</h1>
          {isDescriptionExpanded ? <ChevronUp /> : <ChevronDown />}
        </button>
        {isDescriptionExpanded && (
          <div className="px-4 pb-4 space-y-4">
            <p>
              Given an array of integers nums and an integer target, return indices
              of the two numbers such that they add up to target.
            </p>
            <div className="space-y-2">
              <h3 className="font-medium">Example 1:</h3>
              <pre className="bg-muted p-2 rounded-md">
                Input: nums = [2,7,11,15], target = 9{"\n"}
                Output: [0,1]{"\n"}
                Explanation: Because nums[0] + nums[1] == 9, we return [0, 1].
              </pre>
            </div>
          </div>
        )}
      </div>

      {/* Code Editor and Output */}
      <div className="flex-1 grid grid-cols-2 gap-4 p-4">
        <div className="flex flex-col space-y-4">
          <div className="flex items-center justify-between">
            <select className="bg-background border rounded-md px-2 py-1">
              <option value="javascript">JavaScript</option>
              <option value="python">Python</option>
              <option value="cpp">C++</option>
              <option value="java">Java</option>
            </select>
            <button
              onClick={handleSubmit}
              className="flex items-center space-x-2 bg-primary text-primary-foreground px-4 py-2 rounded-md hover:bg-primary/90"
            >
              <Play className="w-4 h-4" />
              <span>Run</span>
            </button>
          </div>
          <div className="flex-1 border rounded-md overflow-hidden">
            <Editor
              height="100%"
              defaultLanguage="javascript"
              theme="vs-dark"
              value={code}
              onChange={(value) => setCode(value || "")}
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
              }}
            />
          </div>
        </div>

        <div className="flex flex-col space-y-4">
          <h2 className="font-medium">Test Cases</h2>
          <div className="flex-1 border rounded-md p-4 space-y-4">
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Test Case 1:</h3>
              <pre className="bg-muted p-2 rounded-md text-sm">
                nums = [2,7,11,15], target = 9
              </pre>
            </div>
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Test Case 2:</h3>
              <pre className="bg-muted p-2 rounded-md text-sm">
                nums = [3,2,4], target = 6
              </pre>
            </div>
            {output && (
              <div className="space-y-2">
                <h3 className="text-sm font-medium">Output:</h3>
                <pre className="bg-muted p-2 rounded-md text-sm">{output}</pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
} 