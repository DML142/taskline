import { describe, expect, it, vi } from "vitest";
import { ProjectApi } from "./project-api";

describe("ProjectApi", () => {
  it("creates a project through the protected request function", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          project: {
            id: "project-1",
            workspaceId: "workspace-1",
            name: "Website Redesign",
            slug: "website-redesign",
            description: "Refresh the marketing site",
            archived: false,
            createdAt: "2026-09-07T00:00:00Z",
            updatedAt: "2026-09-07T00:00:00Z",
          },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      ),
    );
    const api = new ProjectApi("/api/v1", request);

    await expect(
      api.create("workspace-1", {
        name: "Website Redesign",
        description: "Refresh the marketing site",
      }),
    ).resolves.toMatchObject({ slug: "website-redesign" });
    expect(request).toHaveBeenCalledWith(
      "/api/v1/projects/workspace-1",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("updates a project by its workspace-local slug", async () => {
    const request = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          project: {
            id: "project-1",
            slug: "website-redesign",
            archived: true,
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const api = new ProjectApi("/api/v1", request);

    await expect(
      api.update("workspace-1", "website-redesign", {
        name: "Website Redesign",
        slug: "website-redesign",
        description: "",
        archived: true,
      }),
    ).resolves.toMatchObject({ archived: true });
    expect(request).toHaveBeenCalledWith(
      "/api/v1/projects/workspace-1/website-redesign",
      expect.objectContaining({ method: "PATCH" }),
    );
  });
});
