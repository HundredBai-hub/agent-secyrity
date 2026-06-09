import { useEffect, useMemo, useState } from "react";
import type { ApprovalItem, AuditFilters, AuditRecord, ConsoleSnapshot, Decision } from "./model";
import { MockConsoleApi, type ConsoleApi } from "./consoleApi";
import "./styles.css";

type TabKey = "overview" | "policies" | "approvals" | "audit";

const tabs: Array<{ key: TabKey; label: string }> = [
  { key: "overview", label: "运行时总览" },
  { key: "policies", label: "策略包" },
  { key: "approvals", label: "审批队列" },
  { key: "audit", label: "审计查询" }
];

const decisionLabels: Record<Decision, string> = {
  allow: "allow",
  record: "record",
  redact: "redact",
  require_approval: "require_approval",
  deny: "deny"
};

export function App({ api = new MockConsoleApi() }: { api?: ConsoleApi }) {
  const [snapshot, setSnapshot] = useState<ConsoleSnapshot | null>(null);
  const [activeTab, setActiveTab] = useState<TabKey>("overview");
  const [auditFilters, setAuditFilters] = useState<AuditFilters>({});
  const [auditRecords, setAuditRecords] = useState<AuditRecord[]>([]);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let cancelled = false;
    api
      .getSnapshot()
      .then((data) => {
        if (cancelled) return;
        setSnapshot(data);
        setAuditRecords(data.auditRecords);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : "控制台数据加载失败");
      });
    return () => {
      cancelled = true;
    };
  }, [api]);

  useEffect(() => {
    let cancelled = false;
    api
      .queryAudit(auditFilters)
      .then((records) => {
        if (!cancelled) setAuditRecords(records);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : "审计查询失败");
      });
    return () => {
      cancelled = true;
    };
  }, [api, auditFilters]);

  const pendingApprovals = useMemo(() => snapshot?.approvals.filter((item) => item.status === "pending") ?? [], [snapshot]);

  async function decideApproval(id: string, decision: "approved" | "rejected") {
    try {
      const decided = await api.decideApproval(id, decision);
      setSnapshot((current) => {
        if (!current) return current;
        return {
          ...current,
          approvals: current.approvals.map((item) => (item.id === id ? decided : item))
        };
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "审批操作失败");
    }
  }

  function updateFilter<K extends keyof AuditFilters>(key: K, value: AuditFilters[K]) {
    setAuditFilters((current) => ({ ...current, [key]: value }));
  }

  if (error) {
    return (
      <main className="console-shell">
        <section className="empty-state" role="alert">
          <h1>控制台加载失败</h1>
          <p>{error}</p>
        </section>
      </main>
    );
  }

  if (!snapshot) {
    return (
      <main className="console-shell">
        <section className="empty-state" aria-busy="true">
          <h1>正在加载控制台</h1>
          <p>正在加载运行时安全数据</p>
        </section>
      </main>
    );
  }

  return (
    <main className="console-shell">
      <aside className="sidebar" aria-label="控制台导航">
        <div className="brand-mark">AS</div>
        <nav className="tab-list" role="tablist" aria-label="控制台视图">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              className={activeTab === tab.key ? "tab-button active" : "tab-button"}
              type="button"
              role="tab"
              aria-selected={activeTab === tab.key}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </aside>

      <section className="workspace">
        <header className="workspace-header">
          <div>
            <p className="eyebrow">{snapshot.environment}</p>
            <h1>Agent 运行时安全控制台</h1>
            <p className="tenant-line">{snapshot.tenantName}</p>
          </div>
          <div className="status-strip" aria-label="运行状态">
            <span className="status-dot" />
            <span>Runtime guard active</span>
            <strong>{pendingApprovals.length} pending</strong>
          </div>
        </header>

        {activeTab === "overview" && <Overview snapshot={snapshot} />}
        {activeTab === "policies" && <PolicyPacks snapshot={snapshot} />}
        {activeTab === "approvals" && <Approvals approvals={snapshot.approvals} onDecide={decideApproval} />}
        {activeTab === "audit" && (
          <AuditView filters={auditFilters} records={auditRecords} onFilterChange={updateFilter} />
        )}
      </section>
    </main>
  );
}

function Overview({ snapshot }: { snapshot: ConsoleSnapshot }) {
  return (
    <section className="view-stack" aria-label="运行时总览">
      <div className="metric-grid">
        {snapshot.metrics.map((metric) => (
          <article className={`metric-tile ${metric.tone}`} key={metric.id}>
            <p>{metric.label}</p>
            <strong>{metric.value.toLocaleString("zh-CN")}</strong>
            <span>{metric.trend}</span>
          </article>
        ))}
      </div>
      <section className="two-column">
        <div className="panel">
          <div className="panel-heading">
            <h2>决策分布</h2>
            <span>今日</span>
          </div>
          <div className="decision-bars">
            {snapshot.decisionStats.map((stat) => (
              <div className="decision-row" key={stat.decision}>
                <span>{decisionLabels[stat.decision]}</span>
                <div className="bar-track">
                  <div className={`bar-fill ${stat.decision}`} style={{ width: `${Math.min(stat.count / 9, 100)}%` }} />
                </div>
                <strong>{stat.count}</strong>
              </div>
            ))}
          </div>
        </div>
        <div className="panel">
          <div className="panel-heading">
            <h2>近期高风险事件</h2>
            <span>Top 4</span>
          </div>
          <ul className="event-list">
            {snapshot.auditRecords.map((record) => (
              <li key={record.id}>
                <span className={`decision-pill ${record.decision}`}>{record.decision}</span>
                <div>
                  <strong>{record.agentId}</strong>
                  <p>{record.reason}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </section>
    </section>
  );
}

function PolicyPacks({ snapshot }: { snapshot: ConsoleSnapshot }) {
  return (
    <section className="view-stack" aria-label="策略包">
      <div className="table-panel">
        <table aria-label="策略包列表">
          <thead>
            <tr>
              <th>策略包</th>
              <th>业务场景</th>
              <th>覆盖能力</th>
              <th>策略数</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {snapshot.policyPacks.map((pack) => (
              <tr key={pack.id}>
                <td>
                  <strong>{pack.name}</strong>
                  <span>{pack.id}</span>
                </td>
                <td>{pack.scenario}</td>
                <td>{pack.coverage}</td>
                <td>{pack.policyCount}</td>
                <td>
                  <span className={pack.enabled ? "state enabled" : "state disabled"}>
                    {pack.enabled ? "enabled" : "disabled"}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function Approvals({
  approvals,
  onDecide
}: {
  approvals: ApprovalItem[];
  onDecide: (id: string, decision: "approved" | "rejected") => void;
}) {
  return (
    <section className="view-stack" aria-label="审批队列">
      <div className="table-panel">
        <table aria-label="审批队列">
          <thead>
            <tr>
              <th>审批单</th>
              <th>Agent / 用户</th>
              <th>任务</th>
              <th>动作</th>
              <th>状态</th>
              <th>处理</th>
            </tr>
          </thead>
          <tbody>
            {approvals.map((approval) => (
              <tr key={approval.id}>
                <td>
                  <strong>{approval.id}</strong>
                  <span>{approval.reason}</span>
                </td>
                <td>
                  {approval.agentId}
                  <span>{approval.userId}</span>
                </td>
                <td>{approval.taskId}</td>
                <td>
                  {approval.toolName}
                  <span>{approval.action}</span>
                </td>
                <td>
                  <span className={`state ${approval.status}`}>{approval.status}</span>
                </td>
                <td className="action-cell">
                  <button
                    type="button"
                    className="approve-button"
                    disabled={approval.status !== "pending"}
                    onClick={() => onDecide(approval.id, "approved")}
                  >
                    批准 <span className="sr-only">{approval.id}</span>
                  </button>
                  <button
                    type="button"
                    className="reject-button"
                    disabled={approval.status !== "pending"}
                    onClick={() => onDecide(approval.id, "rejected")}
                  >
                    拒绝 <span className="sr-only">{approval.id}</span>
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function AuditView({
  filters,
  records,
  onFilterChange
}: {
  filters: AuditFilters;
  records: AuditRecord[];
  onFilterChange: <K extends keyof AuditFilters>(key: K, value: AuditFilters[K]) => void;
}) {
  return (
    <section className="view-stack" aria-label="审计查询">
      <div className="filter-bar">
        <label>
          Agent ID
          <input value={filters.agentId ?? ""} onChange={(event) => onFilterChange("agentId", event.target.value)} />
        </label>
        <label>
          用户 ID
          <input value={filters.userId ?? ""} onChange={(event) => onFilterChange("userId", event.target.value)} />
        </label>
        <label>
          任务 ID
          <input value={filters.taskId ?? ""} onChange={(event) => onFilterChange("taskId", event.target.value)} />
        </label>
        <label>
          决策
          <select
            value={filters.decision ?? ""}
            onChange={(event) => onFilterChange("decision", event.target.value as Decision | "")}
          >
            <option value="">全部</option>
            {Object.keys(decisionLabels).map((decision) => (
              <option value={decision} key={decision}>
                {decision}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="table-panel">
        <table aria-label="审计记录">
          <thead>
            <tr>
              <th>审计 ID</th>
              <th>Agent / 用户</th>
              <th>任务</th>
              <th>资源</th>
              <th>决策</th>
              <th>原因</th>
            </tr>
          </thead>
          <tbody>
            {records.map((record) => (
              <tr key={record.id}>
                <td>{record.id}</td>
                <td>
                  {record.agentId}
                  <span>{record.userId}</span>
                </td>
                <td>{record.taskId}</td>
                <td>
                  {record.resource}
                  <span>{record.action}</span>
                </td>
                <td>
                  <span className={`decision-pill ${record.decision}`}>{record.decision}</span>
                </td>
                <td>{record.reason}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
