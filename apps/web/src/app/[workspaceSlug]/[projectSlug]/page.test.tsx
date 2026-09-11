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
let workspaceRole: "OWNER" | "ADMIN" | "MEMBER" | "VIEWER";
let currentUserId: string;
let updateResponder:
  ((url: string, init: RequestInit) => Response | Promise<Response>) | null;
let listResponder: ((url: string) => Response | Promise<Response>) | null;
let createResponder:
  ((url: string, init: RequestInit) => Response | Promise<Response>) | null;

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
              role: workspaceRole,
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
        if (createResponder) return createResponder(url, init);
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
        if (updateResponder) return updateResponder(url, init);
        if (rejectNextUpdate) {
          return new Response(
            JSON.stringify({
              error: { message: "You cannot move this issue." },
            }),
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
      if (listResponder) return listResponder(url);
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
  useAuth: () => ({ protectedRequest, user: { id: currentUserId } }),
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
    workspaceRole = "OWNER";
    currentUserId = "owner-1";
    updateResponder = null;
    listResponder = null;
    createResponder = null;
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

  it("removes a moved card that no longer matches the active status filter", async () => {
    listResponder = (url) =>
      new Response(
        JSON.stringify({
          issues: url.includes("status=TODO")
            ? responseIssues.filter((item) => item.status === "TODO")
            : responseIssues,
        }),
      );
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByText("Ship the redesign");
    await user.selectOptions(screen.getByLabelText("Filter status"), "TODO");
    await waitFor(() =>
      expect(
        within(screen.getByLabelText("In progress")).queryByText(
          "Review the copy",
        ),
      ).toBeNull(),
    );

    const card = screen.getByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(card, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });

    await waitFor(() =>
      expect(screen.queryByText("Ship the redesign")).toBeNull(),
    );
  });

  it("restores a rejected move only when the original card matches active filters", async () => {
    listResponder = (url) =>
      new Response(
        JSON.stringify({
          issues: url.includes("status=TODO")
            ? responseIssues.filter((item) => item.status === "TODO")
            : responseIssues,
        }),
      );
    rejectNextUpdate = true;
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByText("Ship the redesign");
    await user.selectOptions(screen.getByLabelText("Filter status"), "TODO");
    await waitFor(() =>
      expect(
        within(screen.getByLabelText("In progress")).queryByText(
          "Review the copy",
        ),
      ).toBeNull(),
    );

    const card = screen.getByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(card, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });

    expect(
      await within(screen.getByLabelText("To do")).findByText(
        "Ship the redesign",
      ),
    ).toBeTruthy();
  });

  it("does not add a created issue that fails active filters", async () => {
    listResponder = () => new Response(JSON.stringify({ issues: [] }));
    createResponder = () =>
      new Response(
        JSON.stringify({
          issue: {
            ...issue("issue-created", "Hidden issue", "TODO", "LOW"),
            assigneeId: "member-2",
          },
        }),
      );
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByRole("heading", { name: "To do" });
    await user.selectOptions(screen.getByLabelText("Filter status"), "TODO");
    await user.selectOptions(screen.getByLabelText("Priority"), "HIGH");
    await user.selectOptions(
      screen.getAllByLabelText("Assignee")[0],
      "owner-1",
    );
    await user.type(screen.getByLabelText("Title"), "Hidden issue");
    await user.click(screen.getByRole("button", { name: "Create issue" }));

    expect(screen.queryByText("Hidden issue")).toBeNull();
  });

  it("uses filters selected while a create request is pending", async () => {
    let resolveCreate: (response: Response) => void = () => undefined;
    listResponder = () => new Response(JSON.stringify({ issues: [] }));
    createResponder = () =>
      new Promise<Response>((resolve) => {
        resolveCreate = resolve;
      });
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByRole("heading", { name: "To do" });
    await user.type(screen.getByLabelText("Title"), "Late create");
    await user.click(screen.getByRole("button", { name: "Create issue" }));
    await user.selectOptions(screen.getByLabelText("Priority"), "HIGH");

    resolveCreate(
      new Response(
        JSON.stringify({
          issue: issue("issue-created", "Late create", "TODO", "LOW"),
        }),
      ),
    );

    await waitFor(() => expect(screen.queryByText("Late create")).toBeNull());
  });

  it("uses filters selected while a move request is pending", async () => {
    let resolveUpdate: (response: Response) => void = () => undefined;
    updateResponder = () =>
      new Promise<Response>((resolve) => {
        resolveUpdate = resolve;
      });
    listResponder = () =>
      new Response(JSON.stringify({ issues: responseIssues }));
    const user = userEvent.setup();
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(card, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });
    await user.selectOptions(screen.getByLabelText("Priority"), "LOW");

    resolveUpdate(
      new Response(
        JSON.stringify({
          issue: { ...responseIssues[0], status: "IN_PROGRESS" },
        }),
      ),
    );

    await waitFor(() =>
      expect(screen.queryByText("Ship the redesign")).toBeNull(),
    );
  });

  it("ignores a stale list response that resolves after a confirmed move", async () => {
    let resolveStaleList: (response: Response) => void = () => undefined;
    let staleListStarted = false;
    const user = userEvent.setup();
    render(<ProjectPage />);

    await screen.findByText("Ship the redesign");
    listResponder = () =>
      new Promise<Response>((resolve) => {
        staleListStarted = true;
        resolveStaleList = resolve;
      });
    await user.selectOptions(screen.getByLabelText("Priority"), "HIGH");
    await waitFor(() => expect(staleListStarted).toBe(true));

    const card = screen.getByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(card, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });
    expect(
      await within(inProgress).findByText("Ship the redesign"),
    ).toBeTruthy();

    resolveStaleList(new Response(JSON.stringify({ issues: responseIssues })));
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(
      within(screen.getByLabelText("To do")).queryByText("Ship the redesign"),
    ).toBeNull();
    expect(within(inProgress).getByText("Ship the redesign")).toBeTruthy();
  });

  it("clears a column highlight only after the drag leaves the column", async () => {
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const target = screen.getByLabelText("In progress");
    const dataTransfer = { setData: vi.fn() };
    fireEvent.dragStart(card, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    expect(target.className).toContain("border-primary");

    const heading = within(target).getByRole("heading", {
      name: "In progress",
    });
    fireEvent.dragLeave(heading, { relatedTarget: heading });
    expect(target.className).toContain("border-primary");
    fireEvent.dragLeave(target, { relatedTarget: document.body });
    expect(target.className).not.toContain("border-primary");
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

    await waitFor(() =>
      expect(protectedRequest).toHaveBeenCalledWith(
        expect.stringContaining("/issues/workspace-1/website"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            title: "Plan the launch",
            description: "Coordinate the team",
            status: "TODO",
            priority: "LOW",
            assigneeId: null,
          }),
        }),
      ),
    );
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
    expect(dataTransfer.setData).toHaveBeenCalledWith(
      "text/plain",
      "issue-todo",
    );
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

  it("does not make an assigned viewer card draggable or send an update", async () => {
    workspaceRole = "VIEWER";
    currentUserId = "viewer-1";
    responseIssues[0].assigneeId = "viewer-1";
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const target = screen.getByLabelText("In progress");
    const dataTransfer = { getData: vi.fn(), setData: vi.fn() };
    expect(card.closest("a")?.getAttribute("draggable")).toBe("false");
    fireEvent.dragStart(card, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    fireEvent.drop(target, { dataTransfer });

    expect(
      protectedRequest.mock.calls.filter(
        ([, init]) => init?.method === "PATCH",
      ),
    ).toHaveLength(0);
  });

  it("allows a member to move only their assigned card", async () => {
    workspaceRole = "MEMBER";
    currentUserId = "member-1";
    responseIssues[0].assigneeId = "member-1";
    responseIssues[1].assigneeId = "member-2";
    render(<ProjectPage />);

    const ownCard = await screen.findByText("Ship the redesign");
    const otherCard = screen.getByText("Review the copy");
    expect(ownCard.closest("a")?.getAttribute("draggable")).toBe("true");
    expect(otherCard.closest("a")?.getAttribute("draggable")).toBe("false");

    const target = screen.getByLabelText("In progress");
    const dataTransfer = { setData: vi.fn() };
    fireEvent.dragStart(ownCard, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    fireEvent.drop(target, { dataTransfer });

    await waitFor(() =>
      expect(
        protectedRequest.mock.calls.filter(
          ([, init]) => init?.method === "PATCH",
        ),
      ).toHaveLength(1),
    );
  });

  it("shows the optimistic card position before its PATCH resolves", async () => {
    let resolveUpdate: (response: Response) => void = () => undefined;
    updateResponder = () =>
      new Promise<Response>((resolve) => {
        resolveUpdate = resolve;
      });
    render(<ProjectPage />);

    const card = await screen.findByText("Ship the redesign");
    const target = screen.getByLabelText("In progress");
    const dataTransfer = { setData: vi.fn() };
    fireEvent.dragStart(card, { dataTransfer });
    fireEvent.dragOver(target, { dataTransfer });
    fireEvent.drop(target, { dataTransfer });

    expect(within(target).getByText("Ship the redesign")).toBeTruthy();
    resolveUpdate(
      new Response(
        JSON.stringify({
          issue: { ...responseIssues[0], status: "IN_PROGRESS" },
        }),
      ),
    );
    await waitFor(() =>
      expect(
        protectedRequest.mock.calls.filter(
          ([, init]) => init?.method === "PATCH",
        ),
      ).toHaveLength(1),
    );
  });

  it("does not allow a second move of a card while its first PATCH is pending", async () => {
    let resolveFirstUpdate: (response: Response) => void = () => undefined;
    let updateCount = 0;
    updateResponder = () => {
      updateCount += 1;
      if (updateCount === 1) {
        return new Promise<Response>((resolve) => {
          resolveFirstUpdate = resolve;
        });
      }
      return new Response(
        JSON.stringify({ issue: { ...responseIssues[0], status: "DONE" } }),
      );
    };
    render(<ProjectPage />);

    const todoCard = await screen.findByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(todoCard, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });

    const pendingCard =
      await within(inProgress).findByText("Ship the redesign");
    expect(pendingCard.closest("a")?.getAttribute("draggable")).toBe("false");

    const done = screen.getByLabelText("Done");
    const secondDrag = { getData: vi.fn(() => ""), setData: vi.fn() };
    fireEvent.dragStart(pendingCard, { dataTransfer: secondDrag });
    fireEvent.dragOver(done, { dataTransfer: secondDrag });
    fireEvent.drop(done, { dataTransfer: secondDrag });

    expect(
      protectedRequest.mock.calls.filter(
        ([, init]) => init?.method === "PATCH",
      ),
    ).toHaveLength(1);
    expect(within(done).queryByText("Ship the redesign")).toBeNull();

    resolveFirstUpdate(
      new Response(
        JSON.stringify({
          issue: { ...responseIssues[0], status: "IN_PROGRESS" },
        }),
      ),
    );
  });

  it("preserves a later successful move when an earlier move is rejected", async () => {
    let rejectFirstMove: (response: Response) => void = () => undefined;
    updateResponder = (url, init) => {
      if (url.endsWith("/issue-todo")) {
        return new Promise<Response>((resolve) => {
          rejectFirstMove = resolve;
        });
      }
      const inputBody = JSON.parse(String(init.body)) as Partial<TestIssue>;
      const current = responseIssues.find((item) =>
        url.endsWith(`/${item.id}`),
      );
      return new Response(
        JSON.stringify({ issue: { ...current, ...inputBody } }),
      );
    };
    render(<ProjectPage />);

    const todoCard = await screen.findByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(todoCard, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });

    const progressCard = screen.getByText("Review the copy");
    const done = screen.getByLabelText("Done");
    fireEvent.dragStart(progressCard, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(done, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(done, { dataTransfer: { setData: vi.fn() } });

    expect(await within(done).findByText("Review the copy")).toBeTruthy();
    rejectFirstMove(
      new Response(
        JSON.stringify({ error: { message: "You cannot move this issue." } }),
        { status: 403 },
      ),
    );

    expect(await screen.findByRole("alert")).toBeTruthy();
    expect(within(done).getByText("Review the copy")).toBeTruthy();
  });

  it("keeps a later drag active while an earlier request settles", async () => {
    let rejectFirstMove: (response: Response) => void = () => undefined;
    updateResponder = (url, init) => {
      if (url.endsWith("/issue-todo")) {
        return new Promise<Response>((resolve) => {
          rejectFirstMove = resolve;
        });
      }
      const inputBody = JSON.parse(String(init.body)) as Partial<TestIssue>;
      const current = responseIssues.find((item) =>
        url.endsWith(`/${item.id}`),
      );
      return new Response(
        JSON.stringify({ issue: { ...current, ...inputBody } }),
      );
    };
    render(<ProjectPage />);

    const todoCard = await screen.findByText("Ship the redesign");
    const inProgress = screen.getByLabelText("In progress");
    fireEvent.dragStart(todoCard, { dataTransfer: { setData: vi.fn() } });
    fireEvent.dragOver(inProgress, { dataTransfer: { setData: vi.fn() } });
    fireEvent.drop(inProgress, { dataTransfer: { setData: vi.fn() } });

    const progressCard = screen.getByText("Review the copy");
    const done = screen.getByLabelText("Done");
    const laterDataTransfer = { getData: vi.fn(() => ""), setData: vi.fn() };
    fireEvent.dragStart(progressCard, { dataTransfer: laterDataTransfer });
    rejectFirstMove(
      new Response(
        JSON.stringify({ error: { message: "You cannot move this issue." } }),
        { status: 403 },
      ),
    );
    await screen.findByRole("alert");
    fireEvent.dragOver(done, { dataTransfer: laterDataTransfer });
    fireEvent.drop(done, { dataTransfer: laterDataTransfer });

    await waitFor(() =>
      expect(
        protectedRequest.mock.calls.filter(
          ([, init]) => init?.method === "PATCH",
        ),
      ).toHaveLength(2),
    );
  });
});
