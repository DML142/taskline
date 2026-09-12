import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const auth = {
  ready: true,
  user: null as { id: string; email: string; name: string } | null,
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => auth,
}));

import Home from "./page";

afterEach(cleanup);

describe("Home", () => {
  it("gives anonymous visitors clear routes to sign in or create an account", () => {
    auth.user = null;
    render(<Home />);

    expect(
      screen.getByRole("heading", { name: "Make the next task clear." }),
    ).toBeTruthy();
    for (const link of screen.getAllByRole("link", { name: "Sign in" })) {
      expect(link.getAttribute("href")).toBe("/login");
    }
    for (const link of screen.getAllByRole("link", {
      name: "Create an account",
    })) {
      expect(link.getAttribute("href")).toBe("/register");
    }
    expect(
      screen.getByRole("link", { name: "Learn more" }).getAttribute("href"),
    ).toBe("/about");
  });

  it("gives a signed-in visitor a direct route into the application", () => {
    auth.user = { id: "user-1", email: "ada@example.com", name: "Ada" };
    render(<Home />);

    expect(screen.getByText("Ada")).toBeTruthy();
    expect(
      screen.getByRole("link", { name: "Open app" }).getAttribute("href"),
    ).toBe("/app");
    expect(screen.queryByRole("link", { name: "Sign in" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Create account" })).toBeNull();
  });
});
