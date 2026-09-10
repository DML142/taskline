import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

type TestIssue = {
  id: string;
  projectId: string;
  title: string;
  description: string;
  status: "TODO" | "IN_PROGRESS" | "DONE";
  priority: "LOW" | "MEDIUM" | "HIGH";
  creatorId: string;
  assigneeId: string | null;
  createdAt: string;
  updatedAt: string;
};

const issue = (
  id: string,
  title: string,
  status: TestIssue["status"],
  priority: TestIssue["priority"] = "MEDIUM",
): TestIssue => ({
  id,
  projectId: "project-1",
  title,
  description: `${title} description`,
  status,
  priority,
  creatorId: "owner-1",
  assigneeId: null,
  createdAt: "2026-09-10T00:00:00Z",
  updatedAt: "2026-09-10T00:00:00Z",
});

let responseIssues: TestIssue[];
let rejectNextUpdate = false;

const protectedRequest = vi.fn(
  async (input: RequestInfo | URL, init?: RequestInit) => {
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
              createdAt: "2026-09-10T00:00:00Z",
              updatedAt: "2026-09-10T00:00:00Z",
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
      return new Response(
        JSON.stringify({
          members: [
            {
              userId: "owner-1",
              email: "owner@example.com",
              name: "Owner",
              role: "OWNER",
              addedByUserId: null,
              addedByName: null,
              createdAt: "2026-09-10T00:00:00Z",
            },
          ],
        }),
      );
    }
    if (url.includes("/issues/workspace-1/website")) {
      if (init?.method === "POST") {
        const inputBody = JSON.parse(String(init.body)) as {
          title: string;
          description: string;
          priority: TestIssue["priority"];
          assigneeId: string | null;
        };
        return new Response(
          JSON.stringify({
            issue: {
              ...issue(
                "issue-created",
                inputBody.title,
                "TODO",
                inputBody.priority,
              ),
              description: inputBody.description,
              assigneeId: inputBody.assigneeId,
            },
          }),
        );
      }
      if (init?.method === "PATCH") {
        if (rejectNextUpdate) {
          return new Response(
            JSON.stringify({ error: { message: "You cannot move this issue." } }),
            { status: 403 },
          );
        }
        const inputBody = JSON.parse(String(init.body)) as Partial<TestIssue>;
        const current = responseIssues.find((item) =>
          url.endsWith(`/${item.id}`),
        );
        return new Response(
          JSON.stringify({ issue: { ...current, ...inputBody } }),
        );
      }
      return new Response(JSON.stringify({ issues: responseIssues }));
    }
    return new Response(null, { status: 404 });
  },
);

vi.mock("next/navigation", () => ({
  useParams: () => ({ workspaceSlug: "platform", projectSlug: "website" }),
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ protectedRequest, user: { id: "owner-1" } }),
}));

import ProjectPage from "./page";

describe("ProjectPage", () => {
  beforeEach(() => {
    responseIssues = [
      issue("issue-todo", "Ship the redesign", "TODO", "HIGH"),
      issue("issue-progress", "Review the copy", "IN_PROGRESS", "MEDIUM"),
      issue("issue-done", "Publish the brief", "DONE", "LOW"),
    ];
    rejectNextUpdate = false;
    protectedRequest.mockClear();
  });

  afterEach(cleanup);

  it("groups issues into the three Kanban columns", async () => {
    render(<ProjectPage />);

    expect(await screen.findByRole("heading", { name: "To do" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "In progress" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Done" })).toBeTruthy();
    expect(screen.getByLabelText("Priority")).toBeTruthy();
    expect(
      await within(screen.getByLabelText("To do")).findByText(
        "Ship the redesign",
      ),
    ).toBeTruthy();
    expect(
      within(screen.getByLabelText("In progress")).getByText("Review the copy"),
    ).toBeTruthy();
    expect(
      within(screen.getByLabelText("Done")).getByText("Publish the brief"),
    ).toBeTruthy();
  });

  it("requests issues filtered by priority", async () => {
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByText("Ship the redesign");
    await user.selectOptions(screen.getByLabelText("Priority"), "HIGH");

    await waitFor(() =>
      expect(protectedRequest).toHaveBeenCalledWith(
        expect.stringContaining("priority=HIGH"),
      ),
    );
  });

  it("adds a quick-created issue to the To do column", async () => {
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByText("Ship the redesign");
    await user.type(screen.getByLabelText("Title"), "Plan the launch");
    await user.type(
      screen.getAllByLabelText("Description")[1],
      "Coordinate the team",
    );
    await user.selectOptions(screen.getByLabelText("Issue priority"), "LOW");
    await user.click(screen.getByRole("button", { name: "Create issue" }));

    expect(
      await within(screen.getByLabelText("To do")).findByText(
        "Plan the launch",
      ),
    ).toBeTruthy();
  });

  it("moves a permitted card optimistically and persists its complete input", async () => {
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const target = screen.getByLabelText("In progress");
    const dataTransfer = { setData: vi.fn() };
    fireEvent.dragStart(card, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    fireEvent.drop(target, { dataTransfer });

    await waitFor(() =>
      expect(protectedRequest).toHaveBeenCalledWith(
        expect.stringContaining("/issues/workspace-1/website/issue-todo"),
        expect.objectContaining({
          method: "PATCH",
          body: JSON.stringify({
            title: "Ship the redesign",
            description: "Ship the redesign description",
            priority: "HIGH",
            assigneeId: null,
            status: "IN_PROGRESS",
          }),
        }),
      ),
    );
    expect(within(target).getByText("Ship the redesign")).toBeTruthy();
    expect(dataTransfer.setData).toHaveBeenCalledWith("text/plain", "issue-todo");
  });

  it("rolls a card back to its original column when moving it is forbidden", async () => {
    rejectNextUpdate = true;
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const todo = screen.getByLabelText("To do");
    const target = screen.getByLabelText("In progress");
    const dataTransfer = { setData: vi.fn() };
    fireEvent.dragStart(card, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    fireEvent.drop(target, { dataTransfer });

    expect((await screen.findByRole("alert")).textContent).toContain(
      "You cannot move this issue.",
    );
    expect(within(todo).getByText("Ship the redesign")).toBeTruthy();
  });
});
