import type { ProtectedRequest } from "./workspace-api";

export type IssueComment = {
  id: string;
  issueId: string;
  authorId: string;
  authorName: string;
  body: string;
  createdAt: string;
  updatedAt: string;
};

export type CommentInput = { body: string };

export class CommentApi {
  constructor(
    private readonly baseURL: string,
    private readonly request: ProtectedRequest,
  ) {}

  async list(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
  ): Promise<IssueComment[]> {
    const response = await this.request(
      this.url(workspaceId, projectSlug, issueId),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load comments");
    return ((await response.json()) as { comments: IssueComment[] }).comments;
  }

  async create(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
    input: CommentInput,
  ): Promise<IssueComment> {
    const response = await this.request(
      this.url(workspaceId, projectSlug, issueId),
      json("POST", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to create comment");
    return ((await response.json()) as { comment: IssueComment }).comment;
  }

  async update(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
    commentId: string,
    input: CommentInput,
  ): Promise<IssueComment> {
    const response = await this.request(
      `${this.url(workspaceId, projectSlug, issueId)}/${commentId}`,
      json("PATCH", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to update comment");
    return ((await response.json()) as { comment: IssueComment }).comment;
  }

  async delete(
    workspaceId: string,
    projectSlug: string,
    issueId: string,
    commentId: string,
  ): Promise<void> {
    const response = await this.request(
      `${this.url(workspaceId, projectSlug, issueId)}/${commentId}`,
      { method: "DELETE" },
    );
    if (!response.ok)
      throw await responseError(response, "Unable to delete comment");
  }

  private url(workspaceId: string, projectSlug: string, issueId: string) {
    return `${this.baseURL}/issues/${workspaceId}/${projectSlug}/${issueId}/comments`;
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
