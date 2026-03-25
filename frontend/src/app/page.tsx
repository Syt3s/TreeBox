"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Header } from "@/components/layout/header"
import { Footer } from "@/components/layout/footer"
import { useAuth } from "@/components/auth/auth-provider"

export default function Home() {
  const router = useRouter()
  const { user } = useAuth()
  const [searchDomain, setSearchDomain] = useState("")

  const handleSearch = () => {
    const domain = searchDomain.trim()
    if (!domain) {
      return
    }
    router.push(`/box/${domain}`)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-sky-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <main className="container mx-auto px-4 py-16">
        <div className="mx-auto max-w-4xl space-y-12">
          <div className="space-y-6 text-center">
            <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-gradient-to-br from-blue-600 to-cyan-500 shadow-lg">
              <span className="text-5xl font-bold text-white">N</span>
            </div>
            <div className="space-y-2">
              <h1 className="text-5xl font-bold text-gray-900 dark:text-gray-100">NekoBox</h1>
              <p className="text-xl text-gray-600 dark:text-gray-400">一个极简、直接的匿名提问箱。</p>
            </div>
            <p className="mx-auto max-w-2xl text-gray-600 dark:text-gray-400">
              没有复杂的社交关系，没有公开点赞和打扰。你只需要分享自己的提问箱链接，就能开始接收匿名问题并自由回答。
            </p>
          </div>

          <Card className="shadow-lg">
            <CardHeader>
              <CardTitle>查找提问箱</CardTitle>
              <CardDescription>输入用户的个性域名，直接访问对方的提问箱页面。</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Input
                  placeholder="输入用户域名，例如 neko"
                  value={searchDomain}
                  onChange={(e) => setSearchDomain(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      handleSearch()
                    }
                  }}
                />
                <Button onClick={handleSearch}>访问</Button>
              </div>
            </CardContent>
          </Card>

          <div className="grid gap-6 md:grid-cols-3">
            <Card className="shadow-md">
              <CardHeader>
                <CardTitle className="text-lg">匿名提问</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  不暴露身份，适合收集真实反馈、问题和留言。
                </p>
              </CardContent>
            </Card>

            <Card className="shadow-md">
              <CardHeader>
                <CardTitle className="text-lg">简洁体验</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  保留提问和回答这条核心路径，使用起来直接，没有多余干扰。
                </p>
              </CardContent>
            </Card>

            <Card className="shadow-md">
              <CardHeader>
                <CardTitle className="text-lg">自由分享</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  复制你的提问箱链接发到任意平台，就能开始收问题。
                </p>
              </CardContent>
            </Card>
          </div>

          {!user && (
            <Card className="bg-gradient-to-br from-sky-50 to-blue-50 shadow-lg dark:from-blue-900/20 dark:to-cyan-900/20">
              <CardContent className="space-y-4 pt-6 text-center">
                <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">开始使用 NekoBox</h2>
                <p className="text-gray-600 dark:text-gray-400">创建你的提问箱，开始接收匿名提问。</p>
                <div className="flex justify-center gap-4">
                  <Link href="/login">
                    <Button variant="outline" size="lg">
                      登录
                    </Button>
                  </Link>
                  <Link href="/register">
                    <Button size="lg">注册</Button>
                  </Link>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </main>

      <Footer />
    </div>
  )
}
