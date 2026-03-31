"use client"

import * as React from "react"
import { Check, ChevronDown } from "lucide-react"

import { cn } from "@/lib/utils"

export type SelectOption = {
  value: string
  label: string
  description?: string
  disabled?: boolean
}

type SelectProps = {
  value: string
  onChange: (value: string) => void
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  className?: string
  triggerClassName?: string
  contentClassName?: string
  size?: "default" | "sm"
}

export function Select({
  value,
  onChange,
  options,
  placeholder = "请选择",
  disabled = false,
  className,
  triggerClassName,
  contentClassName,
  size = "default",
}: SelectProps) {
  const [open, setOpen] = React.useState(false)
  const containerRef = React.useRef<HTMLDivElement | null>(null)
  const listboxId = React.useId()

  const selectedOption = React.useMemo(
    () => options.find((option) => option.value === value),
    [options, value]
  )

  React.useEffect(() => {
    if (!open) {
      return
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false)
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false)
      }
    }

    document.addEventListener("mousedown", handlePointerDown)
    document.addEventListener("keydown", handleEscape)

    return () => {
      document.removeEventListener("mousedown", handlePointerDown)
      document.removeEventListener("keydown", handleEscape)
    }
  }, [open])

  const triggerSizeClassName =
    size === "sm"
      ? "h-10 rounded-2xl px-3.5 text-sm"
      : "h-11 rounded-[1.4rem] px-4 text-sm"

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxId}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
        className={cn(
          "relative flex w-full items-center justify-between gap-3 border border-cyan-200/80 bg-white/90 text-left text-slate-900 shadow-[0_10px_30px_-18px_rgba(6,182,212,0.55)] transition duration-200",
          "backdrop-blur-sm hover:border-cyan-300 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400/60 focus-visible:ring-offset-2",
          "disabled:cursor-not-allowed disabled:opacity-60 dark:border-cyan-900/60 dark:bg-slate-950/90 dark:text-slate-100 dark:hover:border-cyan-700",
          open && "border-cyan-400 bg-white shadow-[0_20px_40px_-24px_rgba(8,145,178,0.75)] dark:border-cyan-600",
          triggerSizeClassName,
          triggerClassName
        )}
      >
        <span className={cn("min-w-0 flex-1 truncate", !selectedOption && "text-slate-400 dark:text-slate-500")}>
          {selectedOption?.label || placeholder}
        </span>

        <span
          className={cn(
            "flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-cyan-50 text-cyan-700 transition dark:bg-cyan-950/60 dark:text-cyan-200",
            open && "rotate-180 bg-cyan-500 text-white dark:bg-cyan-500 dark:text-white"
          )}
        >
          <ChevronDown className="h-4 w-4" />
        </span>
      </button>

      {open ? (
        <div
          className={cn(
            "absolute left-0 right-0 top-full z-50 mt-2 overflow-hidden rounded-[1.6rem] border border-cyan-100 bg-white/96 p-2 shadow-[0_28px_60px_-24px_rgba(14,116,144,0.55)] backdrop-blur-xl dark:border-cyan-950/70 dark:bg-slate-950/96",
            contentClassName
          )}
        >
          <div id={listboxId} role="listbox" className="max-h-72 overflow-y-auto">
            {options.length === 0 ? (
              <div className="rounded-2xl px-3 py-3 text-sm text-slate-500 dark:text-slate-400">暂无可选项</div>
            ) : (
              options.map((option) => {
                const isSelected = option.value === value

                return (
                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    disabled={option.disabled}
                    onClick={() => {
                      if (option.disabled) {
                        return
                      }
                      onChange(option.value)
                      setOpen(false)
                    }}
                    className={cn(
                      "mb-1 flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-left transition last:mb-0",
                      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400/60",
                      option.disabled && "cursor-not-allowed opacity-50",
                      isSelected
                        ? "bg-gradient-to-r from-cyan-500 to-sky-500 text-white shadow-[0_12px_28px_-18px_rgba(14,165,233,0.9)]"
                        : "text-slate-700 hover:bg-cyan-50/90 hover:text-slate-950 dark:text-slate-200 dark:hover:bg-cyan-950/40 dark:hover:text-white"
                    )}
                  >
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{option.label}</div>
                      {option.description ? (
                        <div
                          className={cn(
                            "mt-1 text-xs leading-5",
                            isSelected ? "text-cyan-50/90" : "text-slate-500 dark:text-slate-400"
                          )}
                        >
                          {option.description}
                        </div>
                      ) : null}
                    </div>

                    <span
                      className={cn(
                        "mt-0.5 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full border",
                        isSelected
                          ? "border-white/40 bg-white/15 text-white"
                          : "border-cyan-100 bg-white text-transparent dark:border-cyan-900 dark:bg-slate-900"
                      )}
                    >
                      <Check className="h-3.5 w-3.5" />
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}
