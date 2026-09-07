import { describe, expect, it, vi } from "vitest";
import { IssueApi } from "./issue-api";

describe("IssueApi", () => {
  it("lists project issues with status and assignee filters", async () => {
    const request = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ issues: [{ id: "issue-1", title: "Ship it" }] }),
        ),
      );
    const api = new IssueApi("/api/v1", request);

    await expect(
      api.list("workspace-1", "website", {
        status: "IN_PROGRESS",
        assigneeId: "user-1",
      }),
    ).resolves.toEqual([{ id: "issue-1", title: "Ship it" }]);

    expect(request).toHaveBeenCalledWith(
      "/api/v1/issues/workspace-1/website?status=IN_PROGRESS&assigneeId=user-1",
    );
  });

  it("creates an issue with the protected request", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ issue: { id: "issue-1", title: "Ship it" } }),
        {
          status: 201,
        },
      ),
    );
    const api = new IssueApi("/api/v1", request);

    await expect(
      api.create("workspace-1", "website", { title: "Ship it" }),
    ).resolves.toMatchObject({
      title: "Ship it",
    });

    expect(request).toHaveBeenCalledWith(
      "/api/v1/issues/workspace-1/website",
      expect.objectContaining({ method: "POST" }),
    );
  });
});
