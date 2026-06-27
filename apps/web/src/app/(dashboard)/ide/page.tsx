"use client";

import { useState } from "react";

export default function IDEPage() {
  const [code, setCode] = useState(`// Welcome to OmniDev IDE
// Start coding here...

function fibonacci(n: number): number {
  if (n <= 1) return n;
  return fibonacci(n - 1) + fibonacci(n - 2);
}

console.log(fibonacci(10));`);

  return (
    <div className="flex h-full">
      {/* File Tree */}
      <div className="w-64 border-r bg-muted/30 p-3">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase mb-3">
          Explorer
        </h3>
        <div className="space-y-1">
          {["src/", "  index.ts", "  utils.ts", "package.json", "tsconfig.json", "README.md"].map(
            (file) => (
              <div
                key={file}
                className="flex items-center gap-2 rounded px-2 py-1 text-sm hover:bg-accent cursor-pointer"
              >
                <span className="text-muted-foreground">
                  {file.endsWith("/") ? "📁" : "📄"}
                </span>
                <span className={file.startsWith("  ") ? "ml-2" : "font-medium"}>
                  {file.trim()}
                </span>
              </div>
            )
          )}
        </div>
      </div>

      {/* Editor */}
      <div className="flex-1 flex flex-col">
        {/* Tabs */}
        <div className="flex border-b bg-muted/30">
          <div className="flex items-center gap-2 px-4 py-2 border-b-2 border-primary text-sm">
            <span>📄</span>
            <span>index.ts</span>
            <button className="ml-2 text-muted-foreground hover:text-foreground">×</button>
          </div>
        </div>

        {/* Code Editor (simplified) */}
        <div className="flex-1 overflow-auto">
          <div className="flex">
            {/* Line numbers */}
            <div className="bg-muted/50 px-4 py-4 text-right select-none">
              {code.split("\n").map((_, i) => (
                <div key={i} className="text-xs text-muted-foreground leading-6">
                  {i + 1}
                </div>
              ))}
            </div>

            {/* Code */}
            <textarea
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className="flex-1 bg-background p-4 font-mono text-sm leading-6 resize-none focus:outline-none"
              spellCheck={false}
            />
          </div>
        </div>

        {/* Terminal */}
        <div className="h-48 border-t bg-black text-green-400 p-4 font-mono text-sm overflow-auto">
          <div className="text-muted-foreground">$ npx ts-node index.ts</div>
          <div className="mt-1">55</div>
          <div className="text-muted-foreground mt-2">$</div>
        </div>
      </div>
    </div>
  );
}
