import { SidebarCard } from "@/components/SidebarCard"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { useState } from "react"

export function AboutCard() {
  const [open, setOpen] = useState(false)

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger asChild>
        <button
          className="w-full text-left text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          aria-label="About Llama of Fate"
        >
          {open ? "▾" : "▸"} About
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <SidebarCard title="About">
          <div className="space-y-2 text-xs text-muted-foreground font-body">
            <p className="font-heading font-semibold text-foreground">
              Llama of Fate v0.1.0
            </p>
            <p>
              by{" "}
              <a
                href="https://github.com/C-Ross"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-foreground"
              >
                C. Ross
              </a>
            </p>
            <p>
              <a
                href="https://github.com/C-Ross/LlamaOfFate"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-foreground"
              >
                github.com/C-Ross/LlamaOfFate
              </a>
            </p>
            <p>A Fate Core RPG with LLM integration</p>
            <p>
              This work is based on Fate Core System, a product of Evil Hat
              Productions, LLC, developed, authored, and edited by Leonard
              Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson,
              Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue,
              and licensed for our use under the{" "}
              <a
                href="https://creativecommons.org/licenses/by/3.0/"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-foreground"
              >
                Creative Commons Attribution 3.0 Unported license
              </a>
              .
            </p>
            <p>
              <a
                href="https://fate-srd.com/"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-foreground"
              >
                Fate Core System Reference Document
              </a>
            </p>
          </div>
        </SidebarCard>
      </CollapsibleContent>
    </Collapsible>
  )
}
