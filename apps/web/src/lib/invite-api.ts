import type { ProtectedRequest, WorkspaceRole } from "./workspace-api";

export type InviteRole = Exclude<WorkspaceRole, "OWNER">;
export type WorkspaceInvite = {
  id: string;
  workspaceId: string;
  email: string;
  role: InviteRole;
  expiresAt: string;
  createdAt: string;
};
export type InvitePreview = {
  workspaceId: string;
  workspaceName: string;
  role: InviteRole;
  expiresAt: string;
};

export class InviteApi {
  constructor(
    private readonly baseURL: string,
    private readonly request: ProtectedRequest,
  ) {}
  async create(
    workspaceId: string,
    input: { email: string; role: InviteRole },
  ): Promise<WorkspaceInvite> {
    const response = await this.request(
      `${this.baseURL}/invitations/workspaces/${workspaceId}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      },
    );
    if (!response.ok) throw new Error("Unable to send invitation");
    return ((await response.json()) as { invite: WorkspaceInvite }).invite;
  }
  async preview(token: string): Promise<InvitePreview> {
    const response = await this.request(
      `${this.baseURL}/workspace-invites/${encodeURIComponent(token)}`,
    );
    if (!response.ok)
      throw new Error(
        response.status === 403
          ? "This invitation is for a different email address."
          : "Invitation is unavailable.",
      );
    return ((await response.json()) as { invite: InvitePreview }).invite;
  }
  async accept(token: string): Promise<void> {
    const response = await this.request(
      `${this.baseURL}/workspace-invites/${encodeURIComponent(token)}/accept`,
      { method: "POST" },
    );
    if (!response.ok)
      throw new Error(
        response.status === 403
          ? "This invitation is for a different email address."
          : "Invitation is unavailable.",
      );
  }
}
