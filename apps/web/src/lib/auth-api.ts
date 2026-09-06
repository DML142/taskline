export type User = { id: string; email: string; name: string };
export type Authentication = { user: User; accessToken: string };
type Fetcher = typeof fetch;

export class AuthApi {
  constructor(
    private readonly baseURL: string,
    private readonly fetcher: Fetcher = fetch,
  ) {}

  async me(accessToken: string): Promise<User> {
    const first = await this.request("/auth/me", accessToken);
    if (first.ok) return (await first.json()).user as User;

    const authentication = await this.refresh();
    const second = await this.request("/auth/me", authentication.accessToken);
    if (!second.ok) throw new Error("Authentication required");
    return (await second.json()).user as User;
  }

  async refresh(): Promise<Authentication> {
    const response = await this.fetcher(`${this.baseURL}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    });
    if (!response.ok) throw new Error("Authentication required");
    return response.json() as Promise<Authentication>;
  }

  async register(
    name: string,
    email: string,
    password: string,
  ): Promise<Authentication> {
    return this.credentials("/auth/register", { name, email, password });
  }

  async login(email: string, password: string): Promise<Authentication> {
    return this.credentials("/auth/login", { email, password });
  }

  async logout(): Promise<void> {
    await this.fetcher(`${this.baseURL}/auth/logout`, {
      method: "POST",
      credentials: "include",
    });
  }

  private async credentials(
    path: string,
    body: Record<string, string>,
  ): Promise<Authentication> {
    const response = await this.fetcher(`${this.baseURL}${path}`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) throw new Error("Authentication failed");
    return response.json() as Promise<Authentication>;
  }

  private request(path: string, accessToken: string) {
    return this.fetcher(`${this.baseURL}${path}`, {
      credentials: "include",
      headers: { Authorization: `Bearer ${accessToken}` },
    });
  }
}
