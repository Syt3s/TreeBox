"use client"

import { useEffect, useState } from "react"
import { Header } from "@/components/layout/header"
import { Footer } from "@/components/layout/footer"

type ChangeLogItem = {
  date: string
  content: string
}

export default function ChangeLogsPage() {
  const [logs, setLogs] = useState<ChangeLogItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const response = await fetch("https://json.n3ko.cc/nekobox-updatehistory/index.json")
        if (!response.ok) {
          throw new Error("加载更新日志失败")
        }
        const result = await response.json()
        if (!cancelled) {
          setLogs(Array.isArray(result?.data) ? result.data : [])
        }
      } catch (fetchError: any) {
        if (!cancelled) {
          setError(fetchError.message || "加载更新日志失败")
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-cyan-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <Header />

      <main className="container mx-auto px-4 py-16">
        <div className="mx-auto max-w-4xl">
          <div className="mb-8 space-y-3">
            <div className="inline-flex rounded-full border border-cyan-200 bg-cyan-100 px-4 py-1 text-sm font-medium text-cyan-800 dark:border-cyan-900/60 dark:bg-cyan-900/30 dark:text-cyan-200">
              开发日志
            </div>
            <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
              NekoBox 开发日记
            </h1>
            <p className="max-w-2xl text-lg leading-8 text-gray-600 dark:text-gray-400">
              这里记录了功能迭代历史和一些开发过程里的说明。现在由前端直接渲染，不再依赖旧的后端模板页。
            </p>
          </div>

          <div className="rounded-[2rem] border border-gray-200 bg-white p-6 shadow-lg dark:border-gray-800 dark:bg-gray-900">
            {loading ? (
              <p className="text-sm text-gray-500 dark:text-gray-400">正在加载更新日志...</p>
            ) : error ? (
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            ) : logs.length > 0 ? (
              <div className="space-y-6">
                {logs.map((log) => (
                  <article key={log.date} className="border-b border-gray-200 pb-6 last:border-b-0 last:pb-0 dark:border-gray-800">
                    <p className="text-sm font-medium text-cyan-600 dark:text-cyan-300">
                      {new Date(log.date).toLocaleDateString()}
                    </p>
                    <div
                      className="mt-2 text-sm leading-7 text-gray-700 dark:text-gray-300"
                      dangerouslySetInnerHTML={{ __html: log.content }}
                    />
                  </article>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">暂无更新日志。</p>
            )}
          </div>
        </div>
      </main>

      <Footer />
    </div>
  )
}