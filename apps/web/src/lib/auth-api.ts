export type User = { id: string; email: string; name: string };
export type Authentication = { user: User; accessToken: string };
export type VerificationRequired = { verificationRequired: true };
type Fetcher = typeof fetch;

export class AuthApi {
  constructor(
    private readonly baseURL: string,
    private readonly fetcher: Fetcher = (...args) => fetch(...args),
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
    if (!response.ok)
      throw await responseError(response, "Authentication required");
    return response.json() as Promise<Authentication>;
  }

  async register(
    name: string,
    email: string,
    password: string,
  ): Promise<VerificationRequired> {
    const response = await this.fetcher(`${this.baseURL}/auth/register`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, password }),
    });
    if (!response.ok)
      throw await responseError(response, "Registration failed");
    return response.json() as Promise<VerificationRequired>;
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

  async verifyEmail(token: string): Promise<void> {
    await this.verificationRequest("/auth/verify-email", { token });
  }

  async resendVerification(email: string): Promise<void> {
    await this.verificationRequest("/auth/resend-verification", { email });
  }

  private async verificationRequest(
    path: string,
    body: Record<string, string>,
  ): Promise<void> {
    const response = await this.fetcher(`${this.baseURL}${path}`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok)
      throw await responseError(response, "Unable to complete verification");
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
    if (!response.ok)
      throw await responseError(response, "Authentication failed");
    return response.json() as Promise<Authentication>;
  }

  private request(path: string, accessToken: string) {
    return this.fetcher(`${this.baseURL}${path}`, {
      credentials: "include",
      headers: { Authorization: `Bearer ${accessToken}` },
    });
  }
}

async function responseError(response: Response, fallback: string) {
  const body = (await response.json().catch(() => null)) as {
    error?: { message?: string };
  } | null;
  return new Error(body?.error?.message ?? fallback);
}
