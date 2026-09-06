import { describe, expect, it, vi } from "vitest";
import { AuthApi } from "./auth-api";

function response(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("AuthApi", () => {
  it("refreshes once and retries a protected request", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(
        response(401, {
          error: {
            code: "unauthenticated",
            message: "Authentication required",
          },
        }),
      )
      .mockResolvedValueOnce(
        response(200, {
          user: { id: "user-1", email: "ada@example.com", name: "Ada" },
          accessToken: "new-token",
        }),
      )
      .mockResolvedValueOnce(
        response(200, {
          user: { id: "user-1", email: "ada@example.com", name: "Ada" },
        }),
      );
    const api = new AuthApi("http://localhost:8080/api/v1", fetcher);

    await api.me("old-token");

    expect(fetcher).toHaveBeenCalledTimes(3);
    expect(fetcher.mock.calls[2][1].headers.Authorization).toBe(
      "Bearer new-token",
    );
  });
});
