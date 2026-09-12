import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn() }),
}));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ ready: true, user: null }),
}));

import Home from "./page";

describe("Home", () => {
  it("gives anonymous visitors clear routes to sign in or create an account", () => {
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
});
