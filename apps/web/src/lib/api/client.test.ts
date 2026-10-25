import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ACTIVE_RUN_STATUSES, api } from "./client";

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify({ data }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("agent API client", () => {
  beforeEach(() => {
    api.setAccessToken(null);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("submits a task to the agent run endpoint", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse({ id: "run-1" }));

    await api.runAgent("agent-1", "Fix the failing test");

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:9090/api/v1/agents/agent-1/run",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ task: "Fix the failing test" }),
      })
    );
  });

  it("uses run-scoped routes for run detail and cancellation", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async () => jsonResponse({ id: "run-1" }));

    await api.getAgentRun("run-1");
    await api.cancelAgentRun("run-1");

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "http://localhost:9090/api/v1/agent-runs/run-1",
      expect.objectContaining({ method: "GET" })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:9090/api/v1/agent-runs/run-1/cancel",
      expect.objectContaining({ method: "POST" })
    );
  });

  it("polls only non-terminal run statuses", () => {
    expect(ACTIVE_RUN_STATUSES).toEqual(["created", "planning", "executing", "waiting_tool"]);
    expect(ACTIVE_RUN_STATUSES).not.toContain("success");
    expect(ACTIVE_RUN_STATUSES).not.toContain("failed");
    expect(ACTIVE_RUN_STATUSES).not.toContain("cancelled");
  });
});
