import { useState, useCallback, type FormEvent } from "react"
import { Input } from "@/components/ui/input"
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

  return (
    <div className="border-t border-border px-6 py-4">
      <form className="mx-auto flex max-w-2xl gap-2" onSubmit={handleSubmit}>
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={placeholder}
          disabled={disabled}
          className="flex-1 bg-input font-body"
          aria-label="Player input"
        />
        <Button
          type="submit"
          disabled={disabled || !text.trim()}
          className="font-heading"
        >
          Send
        </Button>
      </form>
    </div>
  )
}
