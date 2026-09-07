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
          },
        ],
      }),
    );
  }
  if (url.endsWith("/projects/workspace-1")) {
    return new Response(
      JSON.stringify({
        projects: [
          {
            id: "project-1",
            name: "Website",
            slug: "website",
            description: "",
            archived: false,
          },
        ],
      }),
    );
  }
  if (url.endsWith("/workspaces/workspace-1/members")) {
    return new Response(JSON.stringify({ members: [] }));
  }
  if (url.includes("/issues/workspace-1/website")) {
    return new Response(
      JSON.stringify({
        issues: [
          {
            id: "issue-1",
            title: "Ship the redesign",
            status: "TODO",
            priority: "HIGH",
            assigneeId: null,
          },
        ],
      }),
    );
  }
  return new Response(null, { status: 404 });
});

vi.mock("next/navigation", () => ({
  useParams: () => ({ workspaceSlug: "platform", projectSlug: "website" }),
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ protectedRequest }),
}));

import ProjectPage from "./page";

describe("ProjectPage", () => {
  it("shows project issues and filtering controls", async () => {
    render(<ProjectPage />);

    expect(await screen.findByText("Ship the redesign")).toBeTruthy();
    expect(screen.getByLabelText("Filter status")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Create issue" })).toBeTruthy();
  });
});
