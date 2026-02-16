import {
  useState,
  useCallback,
  useRef,
  useEffect,
  type FormEvent,
  type KeyboardEvent,
} from "react"
import { Button } from "@/components/ui/button"

interface ChatInputProps {
  onSend: (text: string) => void
  disabled?: boolean
  placeholder?: string
}

export function ChatInput({
  onSend,
  disabled = false,
  placeholder = "What do you do?",
}: ChatInputProps) {
  const [text, setText] = useState("")
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const prevDisabledRef = useRef(disabled)

  // Auto-focus when input becomes enabled after being disabled
  useEffect(() => {
    if (prevDisabledRef.current && !disabled) {
      textareaRef.current?.focus()
    }
    prevDisabledRef.current = disabled
  }, [disabled])

  const handleSubmit = useCallback(
    (e: FormEvent) => {
      e.preventDefault()
      const trimmed = text.trim()
      if (!trimmed || disabled) return
      onSend(trimmed)
      setText("")
    },
    [text, disabled, onSend],
  )

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault()
        const trimmed = text.trim()
        if (!trimmed || disabled) return
        onSend(trimmed)
        setText("")
      }
    },
    [text, disabled, onSend],
  )

  return (
    <div className="border-t border-border px-6 py-4">
      <form className="mx-auto flex max-w-2xl gap-2" onSubmit={handleSubmit}>
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          rows={2}
          className="flex-1 resize-none rounded-md border border-input bg-input px-3 py-2 text-sm font-body ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          aria-label="Player input"
        />
        <Button
          type="submit"
          disabled={disabled || !text.trim()}
          className="font-heading self-end"
        >
          Send
        </Button>
      </form>
      <p className="mx-auto max-w-2xl mt-1 text-xs text-muted-foreground">
        Press Enter to send, Shift+Enter for a new line
      </p>
    </div>
  )
}
