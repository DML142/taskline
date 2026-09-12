import { describe, expect, it, vi } from "vitest";
import { CommentApi } from "./comment-api";

describe("CommentApi", () => {
  it("creates a comment below its issue", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ comment: { id: "comment-1", body: "Ready." } }),
        {
          status: 201,
        },
      ),
    );
    const api = new CommentApi("/api/v1", request);

    await expect(
      api.create("workspace-1", "website", "issue-1", { body: "Ready." }),
    ).resolves.toMatchObject({
      id: "comment-1",
    });

    expect(request).toHaveBeenCalledWith(
      "/api/v1/issues/workspace-1/website/issue-1/comments",
      expect.objectContaining({ method: "POST" }),
    );
  });
});
