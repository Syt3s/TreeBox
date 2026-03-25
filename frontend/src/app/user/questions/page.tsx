"use client"

import { useCallback, useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { useToast } from "@/components/ui/toast"
import { useAuth } from "@/components/auth/auth-provider"
import { api } from "@/lib/api"
import type { Question } from "@/types"

export default function UserQuestionsPage() {
  const router = useRouter()
  const { toast } = useToast()
  const { user, loading: authLoading } = useAuth()

  const [questions, setQuestions] = useState<Question[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)

  const loadQuestions = useCallback(
    async (cursor?: string) => {
      if (!user) {
        return
      }

      if (cursor) {
        setLoadingMore(true)
      } else {
        setLoading(true)
      }

      try {
        const response = await api.user.questions.list({
          page_size: 20,
          cursor,
        })

        if (response.success) {
          setQuestions((prev) => (cursor ? [...prev, ...response.questions] : response.questions))
          setNextCursor(response.next_cursor || null)
          setHasMore(Boolean(response.next_cursor))
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : "加载问题失败，请稍后重试"
        toast(message, "error")
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [toast, user]
  )

  useEffect(() => {
    if (authLoading) {
      return
    }

    if (!user) {
      router.replace("/login?to=/user/questions")
      return
    }

    void loadQuestions()
  }, [authLoading, loadQuestions, router, user])

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  if (authLoading || loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
        <div className="flex min-h-[60vh] items-center justify-center">
          <div className="text-center">
            <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
            <p className="mt-4 text-gray-600 dark:text-gray-400">加载中...</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <div className="container mx-auto px-4 py-8">
        <div className="mx-auto max-w-4xl space-y-6">
          <Card className="shadow-lg">
            <CardHeader>
              <CardTitle>问题管理</CardTitle>
              <p className="text-sm text-gray-600 dark:text-gray-400">管理你收到的所有提问。</p>
            </CardHeader>
          </Card>

          {questions.length === 0 ? (
            <Card className="shadow-md">
              <CardContent className="py-12 text-center">
                <p className="text-gray-600 dark:text-gray-400">暂时还没有收到提问。</p>
                <p className="mt-2 text-sm text-gray-500 dark:text-gray-500">
                  分享你的提问箱链接，让更多人来提问。
                </p>
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={() => router.push(`/box/${user?.domain}`)}
                >
                  查看我的提问箱
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {questions.map((question) => (
                <Card key={question.id} className="shadow-md">
                  <CardContent className="pt-6">
                    <div className="mb-4">
                      <div className="mb-2 flex items-center justify-between">
                        <div className="text-xs text-gray-500">{formatDate(question.created_at)}</div>
                        <div className="flex items-center gap-2">
                          {question.is_private && <Badge variant="secondary">私密</Badge>}
                          {question.answer && <Badge variant="default">已回答</Badge>}
                        </div>
                      </div>
                      <p className="text-gray-900 dark:text-gray-100">{question.content}</p>
                    </div>

                    {question.answer && (
                      <>
                        <Separator className="my-4" />
                        <div className="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
                          <p className="mb-2 text-sm font-medium text-blue-800 dark:text-blue-200">回答</p>
                          <p className="text-gray-900 dark:text-gray-100">{question.answer}</p>
                        </div>
                      </>
                    )}

                    <div className="mt-4 flex justify-end">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => router.push(`/box/${user?.domain}/${question.id}`)}
                      >
                        {question.answer ? "查看或编辑回答" : "去回答"}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}

              {hasMore && (
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => void loadQuestions(nextCursor || undefined)}
                  disabled={loadingMore}
                >
                  {loadingMore ? "加载中..." : "加载更多"}
                </Button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
