import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { App } from "./App";
import { MockConsoleApi } from "./consoleApi";

describe("Operator Console", () => {
  it("renders the core security operations views", async () => {
    render(<App api={new MockConsoleApi()} />);

    expect(await screen.findByRole("heading", { name: "Agent 运行时安全控制台" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "运行时总览" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "策略包" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "审批队列" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "审计查询" })).toBeInTheDocument();
    expect(screen.getByText("今日运行时事件")).toBeInTheDocument();
  });

  it("filters audit records from the audit view", async () => {
    const user = userEvent.setup();
    render(<App api={new MockConsoleApi()} />);

    await user.click(await screen.findByRole("tab", { name: "审计查询" }));
    await user.type(screen.getByLabelText("Agent ID"), "agent-code");
    await user.selectOptions(screen.getByLabelText("决策"), "deny");

    const table = screen.getByRole("table", { name: "审计记录" });
    expect(within(table).getByText("audit-9001")).toBeInTheDocument();
    expect(within(table).queryByText("audit-9002")).not.toBeInTheDocument();
  });

  it("approves a pending approval from the queue", async () => {
    const user = userEvent.setup();
    render(<App api={new MockConsoleApi()} />);

    await user.click(await screen.findByRole("tab", { name: "审批队列" }));
    await user.click(screen.getByRole("button", { name: "批准 approval-1001" }));

    const row = screen.getByRole("row", { name: /approval-1001/ });
    expect(within(row).getByText("approved")).toBeInTheDocument();
  });
});
