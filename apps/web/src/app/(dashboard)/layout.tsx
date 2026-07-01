"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/stores/auth-store";

const navigation = [
  { name: "Chat", href: "/chat", icon: "💬" },
  { name: "Agents", href: "/agent", icon: "🤖" },
  { name: "Knowledge", href: "/knowledge", icon: "📚" },
  { name: "IDE", href: "/ide", icon: "💻" },
  { name: "Workflow", href: "/workflow", icon: "⚡" },
];

const bottomNavigation = [
  { name: "Settings", href: "/settings", icon: "⚙️" },
];

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isAuthenticated, logout, fetchProfile } = useAuthStore();

  // Check auth on mount and fetch profile if needed
  useEffect(() => {
    const token = localStorage.getItem("access_token");
    if (!token) {
      router.push("/login");
      return;
    }

    // If we have a token but no user, fetch profile
    if (!user) {
      fetchProfile().catch(() => {
        router.push("/login");
      });
    }
  }, [user, router, fetchProfile]);

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="flex w-60 flex-col border-r bg-muted/30">
        {/* Logo */}
        <div className="flex h-14 items-center gap-2 border-b px-4">
          <div className="h-7 w-7 rounded-md bg-primary flex items-center justify-center">
            <span className="text-sm font-bold text-primary-foreground">O</span>
          </div>
          <span className="font-semibold">OmniDev</span>
        </div>

        {/* Main Navigation */}
        <nav className="flex-1 space-y-1 p-3">
          {navigation.map((item) => {
            const isActive = pathname.startsWith(item.href);
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <span className="text-lg">{item.icon}</span>
                {item.name}
              </Link>
            );
          })}
        </nav>

        {/* Bottom Navigation */}
        <div className="border-t p-3 space-y-1">
          {bottomNavigation.map((item) => {
            const isActive = pathname.startsWith(item.href);
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )}
              >
                <span className="text-lg">{item.icon}</span>
                {item.name}
              </Link>
            );
          })}

          {/* User */}
          <div className="flex items-center gap-3 rounded-lg px-3 py-2">
            <Avatar className="h-8 w-8">
              <AvatarFallback className="text-xs">
                {user?.nickname?.[0]?.toUpperCase() || "U"}
              </AvatarFallback>
            </Avatar>
            <div className="flex-1 truncate">
              <p className="text-sm font-medium truncate">{user?.nickname || "User"}</p>
              <p className="text-xs text-muted-foreground truncate">{user?.email}</p>
            </div>
            <button
              onClick={logout}
              className="text-muted-foreground hover:text-foreground"
              title="Sign out"
            >
              ↗
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">{children}</main>
    </div>
  );
}
