import Link from "next/link"
import { supabase } from "@/lib/supabase"

const { data: problems } = await supabase
  .from('problems')
  .select('*')
  .is('deleted_at', null)
  .order('created_at', { ascending: false });

export default function ProblemsPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Problems</h1>
      <div className="rounded-lg border">
        <div className="grid grid-cols-12 border-b px-6 py-3 font-medium">
          <div className="col-span-6">Title</div>
          <div className="col-span-4">Difficulty</div>
          <div className="col-span-2">Solve</div>
        </div>
        {problems.map((problem) => (
          <div
            key={problem.id}
            className="grid grid-cols-12 items-center px-6 py-4 hover:bg-muted/50"
          >
            <div className="col-span-6 font-medium">{problem.title}</div>
            <div className="col-span-4">
              <span
                className={`rounded-full px-2 py-1 text-xs font-medium ${
                  problem.difficulty === "Easy"
                    ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                    : problem.difficulty === "Medium"
                    ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
                    : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300"
                }`}
              >
                {problem.difficulty}
              </span>
            </div>
            <div className="col-span-2">
              <Link
                href={`/problems/${problem.id}`}
                className="text-primary hover:underline"
              >
                Solve →
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
} 