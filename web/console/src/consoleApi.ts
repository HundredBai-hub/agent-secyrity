import { initialSnapshot } from "./mockData";
import type { ApprovalItem, ApprovalStatus, AuditFilters, AuditRecord, ConsoleSnapshot } from "./model";

export interface ConsoleApi {
  getSnapshot(): Promise<ConsoleSnapshot>;
  queryAudit(filters: AuditFilters): Promise<AuditRecord[]>;
  decideApproval(id: string, decision: Extract<ApprovalStatus, "approved" | "rejected">): Promise<ApprovalItem>;
}

export class MockConsoleApi implements ConsoleApi {
  private snapshot: ConsoleSnapshot;

  constructor(snapshot: ConsoleSnapshot = initialSnapshot) {
    this.snapshot = structuredClone(snapshot);
  }

  async getSnapshot(): Promise<ConsoleSnapshot> {
    return structuredClone(this.snapshot);
  }

  async queryAudit(filters: AuditFilters): Promise<AuditRecord[]> {
    return this.snapshot.auditRecords.filter((record) => {
      if (filters.tenantId && record.tenantId !== filters.tenantId) return false;
      if (filters.agentId && !record.agentId.includes(filters.agentId)) return false;
      if (filters.userId && !record.userId.includes(filters.userId)) return false;
      if (filters.taskId && !record.taskId.includes(filters.taskId)) return false;
      if (filters.decision && record.decision !== filters.decision) return false;
      return true;
    });
  }

  async decideApproval(id: string, decision: Extract<ApprovalStatus, "approved" | "rejected">): Promise<ApprovalItem> {
    const approval = this.snapshot.approvals.find((item) => item.id === id);
    if (!approval) {
      throw new Error(`approval ${id} not found`);
    }
    approval.status = decision;
    return structuredClone(approval);
  }
}
