import { describe, expect, it } from "vitest";
import { MockConsoleApi } from "./consoleApi";

describe("MockConsoleApi", () => {
  it("filters audit records by agent, task and decision", async () => {
    const api = new MockConsoleApi();

    const records = await api.queryAudit({
      agentId: "agent-code",
      taskId: "incident",
      decision: "require_approval"
    });

    expect(records).toHaveLength(1);
    expect(records[0].id).toBe("audit-9004");
  });

  it("updates approval decision state", async () => {
    const api = new MockConsoleApi();

    const decided = await api.decideApproval("approval-1001", "approved");
    const snapshot = await api.getSnapshot();

    expect(decided.status).toBe("approved");
    expect(snapshot.approvals.find((item) => item.id === "approval-1001")?.status).toBe("approved");
  });
});
