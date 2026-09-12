import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const replace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ ready: true, user: null }),
}));

import AppPage from "./page";

describe("AppPage", () => {
  it("sends an anonymous visitor to sign in", async () => {
    render(<AppPage />);

    await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  });
});
