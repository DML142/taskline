import { describe, expect, it, vi } from "vitest";
import { WorkspaceApi } from "./workspace-api";

describe("WorkspaceApi", () => {
  it("lists workspaces through the protected request function", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          workspaces: [
            {
              id: "workspace-1",
              name: "Platform",
              slug: "platform",
              role: "OWNER",
              createdAt: "2026-09-07T00:00:00Z",
              updatedAt: "2026-09-07T00:00:00Z",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const api = new WorkspaceApi("/api/v1", request);

    await expect(api.list()).resolves.toMatchObject([
      { id: "workspace-1", name: "Platform", role: "OWNER" },
    ]);
    expect(request).toHaveBeenCalledWith("/api/v1/workspaces");
  });
});
