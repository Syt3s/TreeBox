"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/components/auth/auth-provider"
import { Footer } from "@/components/layout/footer"
import { Header } from "@/components/layout/header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

function buildPixelShellURL() {
  const apiBaseURL = process.env.NEXT_PUBLIC_API_URL?.trim().replace(/\/$/, "") || ""
  if (!apiBaseURL) {
    return "/pixel-shell.html"
  }
  return `/pixel-shell.html?apiBase=${encodeURIComponent(apiBaseURL)}`
}

export default function PixelPage() {
  const router = useRouter()
  const { user, loading } = useAuth()
  const shellURL = buildPixelShellURL()

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login?to=/pixel")
    }
  }, [loading, router, user])

  if (loading || !user) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-white to-cyan-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
        <Header />
        <main className="container mx-auto flex min-h-[calc(100vh-16rem)] items-center justify-center px-4 py-16">
          <Card className="w-full max-w-lg shadow-xl">
            <CardHeader className="text-center">
              <CardTitle>Loading Pixel</CardTitle>
              <CardDescription>Checking your login status and preparing the pixel board.</CardDescription>
            </CardHeader>
          </Card>
        </main>
        <Footer />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-white to-cyan-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Header />

      <main className="container mx-auto px-4 py-10">
        <div className="mb-6 space-y-2">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Pixel</h1>
          <p className="max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-400">
            This page is now served by the frontend. The backend only keeps the authenticated pixel proxy.
          </p>
        </div>

        <Card className="overflow-hidden border-emerald-100 shadow-xl dark:border-gray-800">
          <CardContent className="p-0">
            <iframe
              title="TreeBox Pixel"
              src={shellURL}
              className="h-[calc(100vh-16rem)] min-h-[720px] w-full border-0 bg-white"
            />
          </CardContent>
        </Card>
      </main>

      <Footer />
    </div>
  )
}
