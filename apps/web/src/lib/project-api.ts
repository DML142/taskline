import type { ProtectedRequest } from "./workspace-api";

export type Project = {
  id: string;
  workspaceId: string;
  name: string;
  slug: string;
  description: string;
  archived: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ProjectInput = Pick<Project, "name" | "description"> & {
  slug?: string;
};

export type ProjectUpdate = Required<ProjectInput> & { archived: boolean };

export class ProjectApi {
  constructor(
    private readonly baseURL: string,
    private readonly request: ProtectedRequest,
  ) {}

  async list(workspaceId: string): Promise<Project[]> {
    const response = await this.request(
      `${this.baseURL}/projects/${workspaceId}`,
    );
    if (!response.ok)
      throw await responseError(response, "Unable to load projects");
    return ((await response.json()) as { projects: Project[] }).projects;
  }

  async create(workspaceId: string, input: ProjectInput): Promise<Project> {
    const response = await this.request(
      `${this.baseURL}/projects/${workspaceId}`,
      json("POST", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to create project");
    return ((await response.json()) as { project: Project }).project;
  }

  async update(
    workspaceId: string,
    slug: string,
    input: ProjectUpdate,
  ): Promise<Project> {
    const response = await this.request(
      `${this.baseURL}/projects/${workspaceId}/${slug}`,
      json("PATCH", input),
    );
    if (!response.ok)
      throw await responseError(response, "Unable to update project");
    return ((await response.json()) as { project: Project }).project;
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
