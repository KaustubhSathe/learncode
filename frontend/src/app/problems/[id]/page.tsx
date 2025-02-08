"use client"

import { useState, useEffect, useRef, useCallback } from "react"
import Editor from "@monaco-editor/react"
import { Play, Loader2 } from "lucide-react"
import { useRouter, usePathname } from 'next/navigation'
import { Problem, Submission, User } from '@/types'
import { loader } from '@monaco-editor/react'
import ReactMarkdown from 'react-markdown'
import { setLoading } from "@/store/auth-slice"
import { setUser } from "@/store/auth-slice"
import { useAppSelector } from "@/store/hooks"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

// Configure Monaco Editor
loader.config({
  paths: {
    vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs'
  }
})

// Configure Monaco Editor features before loading
loader.init().then(monaco => {
  // JavaScript configuration
  monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
    noSemanticValidation: false,
    noSyntaxValidation: false,
  })

  monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
    target: monaco.languages.typescript.ScriptTarget.ESNext,
    allowNonTsExtensions: true,
    moduleResolution: monaco.languages.typescript.ModuleResolutionKind.NodeJs,
    module: monaco.languages.typescript.ModuleKind.CommonJS,
    noEmit: true,
    esModuleInterop: true,
    jsx: monaco.languages.typescript.JsxEmit.React,
    allowJs: true,
    typeRoots: ["node_modules/@types"]
  })

  // Add type definitions
  fetch('https://unpkg.com/@types/node/index.d.ts').then(response => response.text()).then(types => {
    monaco.languages.typescript.javascriptDefaults.addExtraLib(
      types,
      'ts:node.d.ts'
    )
  })

  // Configure Java language features
  monaco.languages.register({ id: 'java' })
  
  // Add Java keywords and snippets
  monaco.languages.setMonarchTokensProvider('java', {
    keywords: [
      'abstract', 'continue', 'for', 'new', 'switch', 'assert', 'default', 
      'goto', 'package', 'synchronized', 'boolean', 'do', 'if', 'private', 
      'this', 'break', 'double', 'implements', 'protected', 'throw', 'byte', 
      'else', 'import', 'public', 'throws', 'case', 'enum', 'instanceof', 
      'return', 'transient', 'catch', 'extends', 'int', 'short', 'try', 
      'char', 'final', 'interface', 'static', 'void', 'class', 'finally', 
      'long', 'strictfp', 'volatile', 'const', 'float', 'native', 'super', 
      'while', 'String', 'System'
    ],
    
    operators: [
      '=', '>', '<', '!', '~', '?', ':', '==', '<=', '>=', '!=',
      '&&', '||', '++', '--', '+', '-', '*', '/', '&', '|', '^', '%',
      '<<', '>>', '>>>', '+=', '-=', '*=', '/=', '&=', '|=', '^=',
      '%=', '<<=', '>>=', '>>>='
    ],
    
    symbols: /[=><!~?:&|+\-*\/\^%]+/,
    
    tokenizer: {
      root: [
        [/[a-z_$][\w$]*/, { 
          cases: {
            '@keywords': 'keyword',
            '@default': 'identifier'
          }
        }],
        [/[A-Z][\w$]*/, 'type.identifier'],
        { include: '@whitespace' },
        [/[{}()\[\]]/, '@brackets'],
        [/[<>](?!@symbols)/, '@brackets'],
        [/@symbols/, { cases: { '@operators': 'operator', '@default': '' }}],
        [/\d*\.\d+([eE][\-+]?\d+)?/, 'number.float'],
        [/0[xX][0-9a-fA-F]+/, 'number.hex'],
        [/\d+/, 'number'],
        [/[;,.]/, 'delimiter'],
        [/"([^"\\]|\\.)*$/, 'string.invalid'],
        [/"/, { token: 'string.quote', bracket: '@open', next: '@string' }]
      ],
      
      string: [
        [/[^\\"]+/, 'string'],
        [/\\./, 'string.escape.invalid'],
        [/"/, { token: 'string.quote', bracket: '@close', next: '@pop' }]
      ],
      
      whitespace: [
        [/[ \t\r\n]+/, 'white'],
        [/\/\*/, 'comment', '@comment'],
        [/\/\/.*$/, 'comment']
      ],
      
      comment: [
        [/[^\/*]+/, 'comment'],
        [/\*\//, 'comment', '@pop'],
        [/[\/*]/, 'comment']
      ]
    }
  })
  
  // Add Java code snippets
  monaco.languages.registerCompletionItemProvider('java', {
    provideCompletionItems: (model, position) => {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      };
      return {
        suggestions: [
          {
            label: 'sout',
            kind: monaco.languages.CompletionItemKind.Snippet,
            insertText: 'System.out.println(${1:})',
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            documentation: 'Print to standard output',
            range: range
          },
          {
            label: 'psvm',
            kind: monaco.languages.CompletionItemKind.Snippet,
            insertText: [
              'public static void main(String[] args) {',
              '\t${1:}',
              '}'
            ].join('\n'),
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            documentation: 'Public static void main',
            range: range
          }
        ]
      }
    }
  })
})

type Tab = 'testcases' | 'result' | 'submissions'

const SUPPORTED_LANGUAGES = [
  { 
    id: 'javascript', 
    name: 'JavaScript',
    defaultCode: '// Write your solution here\n'
  },
  { 
    id: 'python', 
    name: 'Python',
    defaultCode: '# Write your solution here\n'
  },
  { 
    id: 'java', 
    name: 'Java',
    defaultCode: `public class Solution {
      public static void main(String[] args) {
          // Write your solution here
      }
  }`
  },
] as const

type Language = typeof SUPPORTED_LANGUAGES[number]['id']

export default function ProblemPage() {
  const router = useRouter()
  const pathname = usePathname()
  const problemId = pathname.split('/').pop()
  const [problem, setProblem] = useState<Problem | null>(null)
  const { loading } = useAppSelector(state => state.auth) as { user: User | null, loading: boolean }
  const [error, setError] = useState<string | null>(null)
  const [code, setCode] = useState('')
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(true)
  const [output, setOutput] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [testInput, setTestInput] = useState("")
  const [expectedOutput, setExpectedOutput] = useState("")
  const [leftWidth, setLeftWidth] = useState(40) // percentage
  const [bottomHeight, setBottomHeight] = useState(30) // percentage
  const horizontalDragRef = useRef<HTMLDivElement>(null)
  const verticalDragRef = useRef<HTMLDivElement>(null)
  const isDraggingRef = useRef<'horizontal' | 'vertical' | null>(null)
  const [activeTab, setActiveTab] = useState<Tab>('testcases')
  const [isRunning, setIsRunning] = useState(false)
  const [currentSubmission, setCurrentSubmission] = useState<Submission | null>(null)
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [selectedLanguage, setSelectedLanguage] = useState<Language>('javascript')
  const [isPolling, setIsPolling] = useState(false)

  useEffect(() => {
    const fetchProblem = async () => {
      try {
        const token = localStorage.getItem('auth_token')
        
        if (!token) {
          router.push('/')
          return
        }

        const response = await fetch(`${process.env.API_URL}/problems/${problemId}`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        })

        if (!response.ok) {
          const errorText = await response.text()
          throw new Error('Failed to fetch problem')
        }

        const data = await response.json()
        console.log(data)
        
        setProblem(data.problem)
        setTestInput(data.problem.input)
        setExpectedOutput(data.problem.output)
        
        // Fetch submissions after problem is loaded
        fetchSubmissions()
      } catch (err) {
        console.error('Fetch error:', err)
        setError(err instanceof Error ? err.message : 'An error occurred')
      } finally {
        setLoading(false)
      }
    }

    fetchProblem()
  }, [problemId, router])

  useEffect(() => {
    const selectedLang = SUPPORTED_LANGUAGES.find(l => l.id === selectedLanguage)
    if (selectedLang) {
      setCode(selectedLang.defaultCode)
    }
  }, [selectedLanguage])

  const handleSubmit = async () => {
    if (!code) return
    
    try {
      setIsSubmitting(true)
      const response = await fetch(`${process.env.API_URL}/submit`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        },
        body: JSON.stringify({
          problem_id: problemId,
          language: selectedLanguage,
          code: code,
          type: 'SUBMIT'
        })
      })
      console.log(response)

      if (!response.ok) {
        const error = await response.json()
        console.log(error)
        throw new Error(error.error || 'Failed to submit code')
      }

      const data = await response.json()
      setActiveTab('submissions')
      await fetchSubmissions() // We'll implement this next
    } catch (err) {
      setOutput(err instanceof Error ? err.message : 'Failed to submit code')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleMouseDown = useCallback((direction: 'horizontal' | 'vertical') => (e: React.MouseEvent) => {
    e.preventDefault()
    isDraggingRef.current = direction
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [])

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDraggingRef.current) return
    
    if (isDraggingRef.current === 'horizontal') {
      
      const newWidth = ((e.clientX) / window.innerWidth) * 100
      setLeftWidth(Math.max(20, newWidth))
    } else {
      const container = verticalDragRef.current?.parentElement?.parentElement
      if (!container) return
      const { top, height } = container.getBoundingClientRect()
      const newHeight = ((e.clientY - top) / height) * 100
      setBottomHeight(Math.max(20, 100 - newHeight))
    }
  }, [])

  const handleMouseUp = useCallback(() => {
    isDraggingRef.current = null
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
  }, [handleMouseMove])

  const pollSubmissionStatus = useCallback(async (problemId: string, submissionId: string) => {
    const response = await fetch(
      `${process.env.API_URL}/submissions?problem_id=${problemId}&submission_id=${submissionId}&type=RUN`,
      {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        }
      }
    )
    if (!response.ok) {
      const msg = await response.json()
      console.log(msg)
      throw new Error('Failed to fetch submission status')
    }
    return await response.json()
  }, [])

  const handleRun = async () => {
    try {
      setIsRunning(true)
      setOutput(null)
      setActiveTab('result')

      // Submit code
      const response = await fetch(`${process.env.API_URL}/submit`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
        },
        body: JSON.stringify({
          problem_id: problemId,
          language: selectedLanguage,
          code: code,
          type: 'RUN',
        })
      })

      if (!response.ok) {
        const msg = await response.json()
        console.log(msg)
        throw new Error('Failed to submit code')
      }
      
      const data = await response.json()
      setCurrentSubmission(data.submission)


      // Poll for results
      while (true) {
        await new Promise(resolve => setTimeout(resolve, 1000)) // Wait 1 second between polls
        const submission = (await pollSubmissionStatus(data.submission.problem_id, data.submission.submission_id)).submissions[0]
        console.log(submission)
        setCurrentSubmission(submission)

        if (submission.status === 'completed' || submission.status === 'error') {
          setOutput(submission.result || 'No output')
          setIsRunning(false)
          break
        }
      }
    } catch (error) {
      setOutput(error instanceof Error ? error.message : 'An error occurred')
      setIsRunning(false)
    }
  }

  const fetchSubmissions = async () => {
    try {
      const response = await fetch(
        `${process.env.API_URL}/submissions?problem_id=${problemId}&type=SUBMIT`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('auth_token')}`
          }
        }
      )

      if (!response.ok) {
        throw new Error('Failed to fetch submissions')
      }

      const data = await response.json()
      setSubmissions(data?.submissions ? data.submissions : [])
      
      // Check if we need to continue polling
      const hasInProgressSubmission = data?.submissions?.some(
        (sub: Submission) => sub.status === 'pending' || sub.status === 'running'
      )
      setIsPolling(hasInProgressSubmission)
    } catch (err) {
      console.error('Failed to fetch submissions:', err)
      setIsPolling(false)
    }
  }

  useEffect(() => {
    let pollInterval: NodeJS.Timeout | null = null;
    
    if (isPolling && activeTab === 'submissions') {
      pollInterval = setInterval(fetchSubmissions, 2000) // Poll every 2 seconds
    }

    return () => {
      if (pollInterval) {
        clearInterval(pollInterval)
      }
    }
  }, [isPolling, activeTab])

  useEffect(() => {
    if (activeTab === 'submissions') {
      fetchSubmissions()
    }
  }, [activeTab, problemId])

  return (
    <div className="flex h-full w-full">
      {/* Left section */}
      <div 
        style={{ width: `${leftWidth}%` }} 
        className="h-full p-4 overflow-y-auto border-r dark:border-gray-800"
      >
        <div className="mb-6">
          <h1 className="text-3xl font-bold">{problem?.title}</h1>
          <div className="mt-2">
            <span className={`px-3 py-1 rounded-full text-sm ${
              problem?.difficulty === 'Easy' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' :
              problem?.difficulty === 'Medium' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' :
              'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            }`}>
              {problem?.difficulty}
            </span>
          </div>
        </div>
        <div className="prose dark:prose-invert max-w-none">
          <ReactMarkdown>{problem?.description || ''}</ReactMarkdown>
        </div>
      </div>

      {/* Horizontal resize handle */}
      <div
        ref={horizontalDragRef}
        className="w-1 hover:w-2 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 cursor-col-resize transition-all"
        onMouseDown={(e) => handleMouseDown('horizontal')(e)}
      />

      {/* Right section */}
      <div className="flex flex-col h-full" style={{ width: `${100 - leftWidth}%`}}>
        {/* Code editor */}
        <div style={{ height: `calc(100% - ${bottomHeight}%)` }} className="w-full flex flex-col relative">
          <div className="bg-gray-100 dark:bg-gray-800 p-2 border-b dark:border-gray-700 flex items-center justify-between">
            <div>
              <Select value={selectedLanguage} onValueChange={(value) => setSelectedLanguage(value as Language)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Select Language" />
                </SelectTrigger>
                <SelectContent>
                  {SUPPORTED_LANGUAGES.map(lang => (
                    <SelectItem key={lang.id} value={lang.id}>
                      {lang.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={handleRun}
                disabled={isRunning}
                className={`px-4 py-2 text-sm font-medium rounded-md 
                  bg-primary text-primary-foreground hover:bg-primary/90
                  disabled:opacity-50 flex items-center gap-2 h-8`}
              >
                {isRunning ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Running...
                  </>
                ) : (
                  'Run'
                )}
              </button>
            <button
              onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-3 py-1 h-8 text-sm bg-green-600 hover:bg-green-700 text-white rounded disabled:opacity-50"
            >
                {isSubmitting ? 'Submitting...' : 'Submit'}
            </button>
            </div>
          </div>
          
            <Editor
            language={selectedLanguage}
            value={code}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
              lineNumbers: 'on',
              automaticLayout: true,
                scrollBeyondLastLine: false,
              tabSize: 2,
              wordWrap: 'on',
              scrollbar: {
                vertical: 'hidden',
                horizontal: 'hidden',
                verticalScrollbarSize: 0,
                horizontalScrollbarSize: 0,
                alwaysConsumeMouseWheel: false
              },
              overviewRulerBorder: false,
              hideCursorInOverviewRuler: true,
              overviewRulerLanes: 0,
              quickSuggestions: true,
              suggestOnTriggerCharacters: true,
              acceptSuggestionOnEnter: "on",
              tabCompletion: "on",
              suggestSelection: "first",
              formatOnPaste: true,
              formatOnType: true,
              autoIndent: "full",
              snippetSuggestions: "inline"
            }}
            className="[&_.monaco-editor]:!overflow-hidden [&_.monaco-editor_.overflow-guard]:!overflow-hidden"
            onChange={(value) => setCode(value || '')}
          />
        </div>

        {/* Vertical resize handle */}
        <div
          ref={verticalDragRef}
          className="h-1 hover:h-2 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 cursor-row-resize transition-all relative z-10"
          onMouseDown={(e) => handleMouseDown('vertical')(e)}
        />

        {/* Test cases section */}
        <div 
          style={{ height: `${bottomHeight}%`, minHeight: '20%' }} 
          className="w-full bg-gray-50 dark:bg-gray-900 border-t dark:border-gray-800 relative flex flex-col"
        >
          {/* Tabs Bar */}
          <div className="border-b dark:border-gray-800">
            <div className="flex">
              <button
                onClick={() => setActiveTab('testcases')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'testcases'
                    ? 'border-primary text-primary dark:text-primary'
                    : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                Test Cases
              </button>
              <button
                onClick={() => setActiveTab('result')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'result'
                    ? 'border-primary text-primary dark:text-primary'
                    : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                Test Result
              </button>
              <button
                onClick={() => setActiveTab('submissions')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'submissions'
                    ? 'border-primary text-primary dark:text-primary'
                    : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                Submissions
              </button>
            </div>
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-y-auto p-4">
            {activeTab === 'testcases' && (
              <div className="space-y-4">
                <div>
                  <h3 className="font-medium mb-2">Example Input:</h3>
                  <pre className="bg-gray-100 dark:bg-gray-800 p-3 rounded-md font-mono whitespace-pre-wrap">
                    {problem?.example_input || 'No example input available'}
                  </pre>
                </div>
                <div>
                  <h3 className="font-medium mb-2">Example Output:</h3>
                  <pre className="bg-gray-100 dark:bg-gray-800 p-3 rounded-md font-mono whitespace-pre-wrap">
                    {problem?.example_output || 'No example output available'}
                  </pre>
                </div>
              </div>
            )}
            {activeTab === 'result' && (
              <div>
                {output ? (
                  <pre className="whitespace-pre-wrap">{output}</pre>
                ) : (
                  <p className="text-gray-500 dark:text-gray-400">Run your code to see the results</p>
                )}
            </div>
            )}
            {activeTab === 'submissions' && (
              <div className="space-y-4">
                {submissions?.length === 0 ? (
                  <p className="text-gray-500 dark:text-gray-400">No submissions yet</p>
                ) : (
                  <div className="space-y-4">
                    {submissions?.map((submission) => (
                      <div 
                        key={submission.submission_id} 
                        className="p-4 border rounded-lg dark:border-gray-700"
                      >
                        <div className="text-sm text-gray-500 mb-2">
                          Submission ID: {submission.submission_id}
                        </div>
                        <div className="flex justify-between items-center mb-2">
                          <div className="flex items-center gap-4">
                            <span className={
                              submission.status === 'completed' ? 'text-green-500' :
                              submission.status === 'error' ? 'text-red-500' :
                              'text-yellow-500'
                            }>
                              Status: {submission.status}
                            </span>
                            <span className="text-sm">
                              Language: {submission.language}
                            </span>
                          </div>
                          <span className="text-sm text-gray-500">
                            {new Date(submission.created_at * 1000).toLocaleString()}
                          </span>
                        </div>
                        <div className="mt-2 p-2 bg-gray-100 dark:bg-gray-800 rounded">
                          <pre className="text-sm overflow-x-auto">
                            <code>{submission.code}</code>
                          </pre>
                        </div>
                        {submission.result && (
                          <pre className="mt-2 p-2 bg-gray-100 dark:bg-gray-800 rounded text-sm text-wrap">
                            <div className="font-medium mb-1">Result:</div>
                            {submission.result}
                          </pre>
                        )}

                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
} 