import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import AboutPage from "./page";

describe("AboutPage", () => {
  it("explains the project and links to its GitHub repository", () => {
    render(<AboutPage />);

    expect(
      screen.getByRole("heading", { name: "About Taskline" }),
    ).toBeTruthy();
    expect(
      screen
        .getByRole("link", { name: "View the project on GitHub" })
        .getAttribute("href"),
    ).toBe("https://github.com/DML142/taskline");
  });
});
