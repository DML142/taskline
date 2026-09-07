import type { ProtectedRequest } from "./workspace-api";

export type IssueStatus = "TODO" | "IN_PROGRESS" | "DONE";
export type IssuePriority = "LOW" | "MEDIUM" | "HIGH";

export type Issue = {
  id: string;
  projectId: string;
  title: string;
  description: string;
  status: IssueStatus;
  priority: IssuePriority;
  creatorId: string;
  assigneeId: string | null;
  createdAt: string;
  updatedAt: string;
};

export type IssueInput = {
  title: string;
  description?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assigneeId?: string | null;
};

export type IssueFilter = {
  status?: IssueStatus;
  assigneeId?: string;
};

export class IssueApi {
  constructor(
    private readonly baseURL: string,
    private readonly request: ProtectedRequest,
  ) {}

  async list(
    workspaceId: string,
    projectSlug: string,
    filter: IssueFilter = {},
  ): Promise<Issue[]> {
    const params = new URLSearchParams();
    if (filter.status) params.set("status", filter.status);
    if (filter.assigneeId) params.set("assigneeId", filter.assigneeId);
    const query = params.size ? `?${params}` : "";
    const response = await this.request(
      `${this.baseURL}/issues/${workspaceId}/${projectSlug}${query}`,
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load issues");
    return ((await response.json()) as { issues: Issue[] }).issues;
  }

  async get(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
  ): Promise<Issue> {
    const response = await this.request(
      `${this.baseURL}/issues/${workspaceId}/${projectSlug}/${issueId}`,
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load issue");
    return ((await response.json()) as { issue: Issue }).issue;
  }

  async create(
    workspaceId: string,
    projectSlug: string,
    input: IssueInput,
  ): Promise<Issue> {
    const response = await this.request(
      `${this.baseURL}/issues/${workspaceId}/${projectSlug}`,
      json("POST", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to create issue");
    return ((await response.json()) as { issue: Issue }).issue;
  }

  async update(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
    input: IssueInput,
  ): Promise<Issue> {
    const response = await this.request(
      `${this.baseURL}/issues/${workspaceId}/${projectSlug}/${issueId}`,
      json("PATCH", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to update issue");
    return ((await response.json()) as { issue: Issue }).issue;
  }
}

function json(method: string, body: unknown): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

async function responseError(response: Response, fallback: string) {
  const body = (await response.json().catch(() => null)) as {
    error?: { message?: string };
  } | null;
  return new Error(body?.error?.message ?? fallback);
}
