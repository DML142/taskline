import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  useParams: () => ({
    workspaceSlug: "platform",
    projectSlug: "website",
    issueId: "issue-1",
  }),
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({
    user: { id: "owner-1" },
    protectedRequest: async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/workspaces"))
        return new Response(
          JSON.stringify({
            workspaces: [
              {
                id: "workspace-1",
                name: "Platform",
                slug: "platform",
                role: "OWNER",
              },
            ],
          }),
        );
      if (url.endsWith("/projects/workspace-1"))
        return new Response(
          JSON.stringify({ projects: [{ id: "project-1", slug: "website" }] }),
        );
      if (url.endsWith("/issues/workspace-1/website/issue-1"))
        return new Response(
          JSON.stringify({
            issue: {
              id: "issue-1",
              title: "Ship the redesign",
              description: "",
              status: "TODO",
              priority: "HIGH",
              assigneeId: null,
            },
          }),
        );
      if (url.endsWith("/workspaces/workspace-1/members"))
        return new Response(JSON.stringify({ members: [] }));
      return new Response(null, { status: 404 });
    },
  }),
}));

import IssuePage from "./page";

describe("IssuePage", () => {
  it("shows the issue and an editable form for a workspace owner", async () => {
    render(<IssuePage />);

    expect(await screen.findByDisplayValue("Ship the redesign")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save issue" })).toBeTruthy();
  });
});
