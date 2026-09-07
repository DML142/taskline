export type WorkspaceRole = "OWNER" | "ADMIN" | "MEMBER" | "VIEWER";

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
};

export type WorkspaceSummary = Workspace & { role: WorkspaceRole };

export type WorkspaceMember = {
  userId: string;
  email: string;
  name: string;
  role: WorkspaceRole;
  createdAt: string;
};

export type ProtectedRequest = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>;

export class WorkspaceApi {
  constructor(
    private readonly baseURL: string,
    private readonly request: ProtectedRequest,
  ) {}

  async list(): Promise<WorkspaceSummary[]> {
    const response = await this.request(`${this.baseURL}/workspaces`);
    if (!response.ok)
      throw await responseError(response, "Unable to load workspaces");
    return ((await response.json()) as { workspaces: WorkspaceSummary[] })
      .workspaces;
  }

  async create(name: string): Promise<WorkspaceSummary> {
    const response = await this.request(
      `${this.baseURL}/workspaces`,
      json("POST", { name }),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to create workspace");
    const body = (await response.json()) as {
      workspace: Workspace;
      role: WorkspaceRole;
    };
    return { ...body.workspace, role: body.role };
  }

  async get(workspaceId: string): Promise<WorkspaceSummary> {
    const response = await this.request(
      `${this.baseURL}/workspaces/${workspaceId}`,
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load workspace");
    const body = (await response.json()) as {
      workspace: Workspace;
      role: WorkspaceRole;
    };
    return { ...body.workspace, role: body.role };
  }

  async members(workspaceId: string): Promise<WorkspaceMember[]> {
    const response = await this.request(
      `${this.baseURL}/workspaces/${workspaceId}/members`,
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load members");
    return ((await response.json()) as { members: WorkspaceMember[] }).members;
  }

  async addMember(
    workspaceId: string,
    email: string,
    role: Exclude<WorkspaceRole, "OWNER">,
  ): Promise<void> {
    const response = await this.request(
      `${this.baseURL}/workspaces/${workspaceId}/members`,
      json("POST", { email, role }),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to add member");
  }

  async updateMemberRole(
    workspaceId: string,
    userId: string,
    role: Exclude<WorkspaceRole, "OWNER">,
  ): Promise<void> {
    const response = await this.request(
      `${this.baseURL}/workspaces/${workspaceId}/members/${userId}`,
      json("PATCH", { role }),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to update member");
  }

  async removeMember(workspaceId: string, userId: string): Promise<void> {
    const response = await this.request(
      `${this.baseURL}/workspaces/${workspaceId}/members/${userId}`,
      { method: "DELETE" },
    );
    if (!response.ok)
      throw await responseError(response, "Unable to remove member");
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
