import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const protectedRequest = vi.fn(async (input: RequestInfo | URL) => {
  const url = String(input);
  if (url.endsWith("/workspaces")) {
    return new Response(
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
    );
  }
  if (url.endsWith("/projects/workspace-1")) {
    return new Response(JSON.stringify({ projects: [] }));
  }
  if (url.endsWith("/workspaces/workspace-1/members")) {
    return new Response(
      JSON.stringify({
        members: [
          {
            userId: "owner-1",
            email: "owner@example.com",
            name: "Owner",
            role: "OWNER",
            createdAt: "2026-09-07T00:00:00Z",
          },
        ],
      }),
    );
  }
  if (url.endsWith("/invitations/workspaces/workspace-1")) {
    return new Response(JSON.stringify({ invites: [] }));
  }
  return new Response(null, { status: 404 });
});

vi.mock("next/navigation", () => ({
  useParams: () => ({ workspaceSlug: "platform" }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ protectedRequest }),
}));

import WorkspaceProjectsPage from "./page";

describe("WorkspaceProjectsPage", () => {
  it("shows member management to workspace owners", async () => {
    render(<WorkspaceProjectsPage />);

    expect(
      await screen.findByRole("button", { name: "Send invitation" }),
    ).toBeTruthy();
    expect(screen.getByText("owner@example.com")).toBeTruthy();
  });
});
