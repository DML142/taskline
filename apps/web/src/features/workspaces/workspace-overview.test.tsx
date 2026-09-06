import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const protectedRequest = vi
  .fn()
  .mockRejectedValue(new Error("Resource not found"));

vi.mock("@/features/auth/auth-context", () => ({
  useAuth: () => ({ protectedRequest }),
}));

import { WorkspaceOverview } from "./workspace-overview";

describe("WorkspaceOverview", () => {
  it("replaces the loading state with an API error", async () => {
    render(<WorkspaceOverview />);

    expect((await screen.findByRole("alert")).textContent).toContain(
      "Resource not found",
    );
    expect(screen.queryByText("Loading workspaces…")).toBeNull();
  });
});
