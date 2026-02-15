import { SidebarCard } from "@/components/SidebarCard"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

function App() {
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Chat Panel — left side */}
      <div className="flex flex-1 flex-col">
        {/* Header */}
        <header className="flex items-center gap-3 border-b border-border px-6 py-4">
          <h1 className="text-2xl font-heading font-bold tracking-widest uppercase text-foreground">
            <span className="text-accent-foreground/60">Llama</span> of <span className="text-primary">Fate</span>
          </h1>
          <Badge variant="outline" className="text-muted-foreground">
            Not Connected
          </Badge>
        </header>

        {/* Message area */}
        <ScrollArea className="flex-1 px-6 py-4">
          <div className="mx-auto max-w-2xl space-y-4">
            <div className="rounded-lg bg-secondary/50 px-4 py-3 text-sm text-muted-foreground italic font-body">
              Connect to the server to begin your adventure...
            </div>
          </div>
        </ScrollArea>

        {/* Input area */}
        <div className="border-t border-border px-6 py-4">
          <form
            className="mx-auto flex max-w-2xl gap-2"
            onSubmit={(e) => e.preventDefault()}
          >
            <Input
              placeholder="What do you do?"
              disabled
              className="flex-1 bg-input font-body"
            />
            <Button disabled className="font-heading">
              Send
            </Button>
          </form>
        </div>
      </div>

      {/* Sidebar — right side */}
      <aside className="hidden w-80 flex-col border-l border-border lg:flex">
        <ScrollArea className="flex-1 p-4">
          <div className="space-y-4">
            <SidebarCard title="Character">
              <p className="text-sm text-muted-foreground font-body">
                No character loaded
              </p>
            </SidebarCard>

            <SidebarCard title="Situation Aspects">
              <p className="text-sm text-muted-foreground font-body">
                None active
              </p>
            </SidebarCard>

            <SidebarCard title="Fate Points">
              <span className="text-2xl font-bold text-accent">—</span>
            </SidebarCard>
          </div>
        </ScrollArea>
      </aside>
    </div>
  )
}

export default App
