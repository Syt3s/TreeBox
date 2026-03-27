"use client"

import * as React from "react"
import { cn } from "@/lib/utils"
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from "lucide-react"

type ToastVariant = "default" | "success" | "error" | "warning" | "info"

interface ToastProps {
  id: string
  message: string
  variant?: ToastVariant
  duration?: number
  onClose: (id: string) => void
}

const Toast = ({ id, message, variant = "default", duration = 3000, onClose }: ToastProps) => {
  React.useEffect(() => {
    const timer = setTimeout(() => onClose(id), duration)
    return () => clearTimeout(timer)
  }, [id, duration, onClose])

  const icons = {
    default: Info,
    success: CheckCircle,
    error: AlertCircle,
    warning: AlertTriangle,
    info: Info,
  }

  const colors = {
    default:
      "border-slate-200 bg-white/95 text-slate-900 dark:border-slate-800 dark:bg-slate-950/95 dark:text-slate-100",
    success:
      "border-emerald-200 bg-emerald-50/95 text-emerald-900 dark:border-emerald-900/50 dark:bg-emerald-950/90 dark:text-emerald-100",
    error:
      "border-sky-200 bg-sky-50/95 text-sky-900 dark:border-sky-900/60 dark:bg-sky-950/90 dark:text-sky-100",
    warning:
      "border-amber-200 bg-amber-50/95 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/90 dark:text-amber-100",
    info:
      "border-cyan-200 bg-cyan-50/95 text-cyan-900 dark:border-cyan-900/50 dark:bg-cyan-950/90 dark:text-cyan-100",
  }

  const iconColors = {
    default: "bg-slate-100 text-slate-700 dark:bg-slate-900 dark:text-slate-200",
    success: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/70 dark:text-emerald-200",
    error: "bg-sky-100 text-sky-700 dark:bg-sky-900/70 dark:text-sky-200",
    warning: "bg-amber-100 text-amber-700 dark:bg-amber-900/70 dark:text-amber-200",
    info: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/70 dark:text-cyan-200",
  }

  const Icon = icons[variant]

  return (
    <div
      className={cn(
        "relative flex w-full items-start gap-3 rounded-2xl border px-4 py-3 shadow-lg backdrop-blur transition-all duration-300 animate-in slide-in-from-top-2",
        colors[variant]
      )}
    >
      <div
        className={cn(
          "flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full",
          iconColors[variant]
        )}
      >
        <Icon className="h-5 w-5" />
      </div>
      <span className="flex-1 text-sm font-medium leading-6">{message}</span>
      <button
        onClick={() => onClose(id)}
        className="rounded-full p-1 transition-colors hover:bg-black/5 dark:hover:bg-white/10"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

interface ToastContextType {
  toast: (message: string, variant?: ToastVariant, duration?: number) => void
}

const ToastContext = React.createContext<ToastContextType | undefined>(undefined)

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = React.useState<Array<{ id: string; message: string; variant: ToastVariant; duration: number }>>([])

  const toast = React.useCallback((message: string, variant: ToastVariant = "default", duration: number = 3000) => {
    const id = Math.random().toString(36).substring(2, 9)
    setToasts((prev) => [...prev, { id, message, variant, duration }])
  }, [])

  const removeToast = React.useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed right-4 top-4 z-50 flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-3">
        {toasts.map((t) => (
          <Toast key={t.id} {...t} onClose={removeToast} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const context = React.useContext(ToastContext)
  if (!context) {
    throw new Error("useToast must be used within a ToastProvider")
  }
  return context
}
