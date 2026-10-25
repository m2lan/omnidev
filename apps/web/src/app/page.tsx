"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";

export default function Home() {
  const router = useRouter();
  const { isAuthenticated, fetchProfile } = useAuthStore();
  useEffect(() => {
    const token = localStorage.getItem("access_token");
    if (token && !isAuthenticated) {
      fetchProfile().catch(() => {});
    }
  }, [isAuthenticated, fetchProfile]);

  const handleGetStarted = (e: React.MouseEvent) => {
    e.preventDefault();
    const token = localStorage.getItem("access_token");
    if (token) {
      router.push("/chat");
    } else {
      router.push("/register");
    }
  };

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
          AI knowledge and coding workbench for teams: multi-model chat, searchable
          knowledge bases, and background agents that run code in Docker sandboxes.
        </p>

        {/* Feature Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-4xl">
          {[
            { icon: "💬", title: "AI Chat", desc: "Multi-model conversations" },
            { icon: "📚", title: "Knowledge", desc: "Document search with citations" },
            { icon: "🤖", title: "Agents", desc: "Sandboxed background tasks" },
            { icon: "🔌", title: "MCP", desc: "External tool integrations" },
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
          <a
            href="#"
            onClick={handleGetStarted}
            className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-8 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90"
          >
            {isAuthenticated ? "Go to Chat" : "Get Started"}
          </a>
          {!isAuthenticated && (
            <Link
              href="/login"
              className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-8 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
            >
              Sign In
            </Link>
          )}
        </div>

        {/* Tech Stack */}
        <div className="flex flex-wrap items-center justify-center gap-6 text-sm text-muted-foreground">
          <span>Go</span>
          <span>•</span>
          <span>Next.js</span>
          <span>•</span>
          <span>PostgreSQL + pgvector</span>
          <span>•</span>
          <span>Redis</span>
          <span>•</span>
          <span>Docker</span>
        </div>
      </div>
    </div>
  );
}
