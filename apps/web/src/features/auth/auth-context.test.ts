import { describe, expect, it, vi } from "vitest";
import {
  createRefreshGate,
  type Authentication,
} from "@/features/auth/auth-context";

describe("createRefreshGate", () => {
  it("shares one refresh operation between concurrent callers", async () => {
    let resolveRefresh: (value: Authentication) => void;
    const refresh = vi.fn(
      () =>
        new Promise<Authentication>((resolve) => {
          resolveRefresh = resolve;
        }),
    );
    const gate = createRefreshGate(refresh);

    const first = gate.refresh();
    const second = gate.refresh();
    expect(refresh).toHaveBeenCalledTimes(1);

    resolveRefresh!({
      user: { id: "user-1", email: "ada@example.com", name: "Ada" },
      accessToken: "renewed",
    });
    await expect(Promise.all([first, second])).resolves.toHaveLength(2);
  });
});
