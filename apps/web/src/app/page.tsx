import Link from "next/link";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-background to-muted/50">
      <div className="container flex flex-col items-center gap-8 text-center">
        {/* Logo */}
        <div className="flex items-center gap-3">
          <div className="h-12 w-12 rounded-xl bg-primary flex items-center justify-center">
            <span className="text-2xl font-bold text-primary-foreground">O</span>
          </div>
          <h1 className="text-4xl font-bold tracking-tight">OmniDev</h1>
        </div>

        {/* Tagline */}
        <p className="max-w-[600px] text-lg text-muted-foreground">
          All-in-One AI Development Platform. Build, deploy, and monitor your applications
          with the power of AI.
        </p>

        {/* Feature Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-4xl">
          {[
            { icon: "💬", title: "AI Chat", desc: "Multi-model conversations" },
            { icon: "🤖", title: "Agents", desc: "Autonomous AI workers" },
            { icon: "📚", title: "RAG", desc: "Knowledge base search" },
            { icon: "💻", title: "IDE", desc: "Online code editor" },
            { icon: "⚡", title: "Workflow", desc: "Visual automation" },
            { icon: "🔌", title: "MCP", desc: "Tool integrations" },
            { icon: "🚀", title: "Deploy", desc: "One-click deployment" },
            { icon: "📊", title: "Monitor", desc: "Observability" },
          ].map((feature) => (
            <div
              key={feature.title}
              className="flex flex-col items-center gap-2 rounded-lg border bg-card p-4 text-card-foreground shadow-sm transition-all hover:shadow-md"
            >
              <span className="text-2xl">{feature.icon}</span>
              <h3 className="font-semibold">{feature.title}</h3>
              <p className="text-xs text-muted-foreground">{feature.desc}</p>
            </div>
          ))}
        </div>

        {/* CTA */}
        <div className="flex gap-4">
          <Link
            href="/register"
            className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-8 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90"
          >
            Get Started
          </Link>
          <Link
            href="/login"
            className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-8 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
          >
            Sign In
          </Link>
        </div>

        {/* Tech Stack */}
        <div className="flex flex-wrap items-center justify-center gap-6 text-sm text-muted-foreground">
          <span>Go</span>
          <span>•</span>
          <span>Next.js</span>
          <span>•</span>
          <span>PostgreSQL</span>
          <span>•</span>
          <span>Redis</span>
          <span>•</span>
          <span>Kafka</span>
          <span>•</span>
          <span>Kubernetes</span>
        </div>
      </div>
    </div>
  );
}
