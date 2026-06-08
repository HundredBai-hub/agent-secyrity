package runtime

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/approval"
	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
	"github.com/HundredBai-hub/agent-secyrity/internal/policypack"
)

type Service struct {
	engine        *policy.Engine
	packStore     policypack.Store
	store         audit.Store
	approvalStore approval.Store
	approvalTTL   time.Duration
}

type Options struct {
	Engine        *policy.Engine
	PolicyPacks   policypack.Store
	AuditStore    audit.Store
	ApprovalStore approval.Store
	ApprovalTTL   time.Duration
}

func NewService(engine *policy.Engine, store audit.Store) *Service {
	return &Service{engine: engine, store: store}
}

func NewServiceWithPolicyPacks(packStore policypack.Store, store audit.Store) *Service {
	return &Service{packStore: packStore, store: store}
}

func NewServiceWithOptions(opts Options) *Service {
	return &Service{
		engine:        opts.Engine,
		packStore:     opts.PolicyPacks,
		store:         opts.AuditStore,
		approvalStore: opts.ApprovalStore,
		approvalTTL:   opts.ApprovalTTL,
	}
}

func (s *Service) Evaluate(ctx context.Context, event domain.RuntimeEvent) (domain.EvaluationResult, error) {
	normalized, err := event.Normalize()
	if err != nil {
		return domain.EvaluationResult{}, &domain.RuntimeEventError{Err: err}
	}
	event = normalized
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.ApprovalID != "" {
		result := s.evaluateApproval(ctx, event)
		return s.appendAudit(ctx, event, result)
	}
	engine := s.engine
	if s.packStore != nil {
		packs, err := s.packStore.ListEnabled(ctx, event.TenantID)
		if err != nil {
			return domain.EvaluationResult{}, fmt.Errorf("list enabled policy packs: %w", err)
		}
		engine = policy.NewEngineFromPacks(packs)
	}
	if engine == nil {
		engine = policy.NewEngine(nil)
	}
	result := engine.Evaluate(event)
	if result.Decision == domain.DecisionRequireApproval && s.approvalStore != nil {
		ttl := s.approvalTTL
		if ttl <= 0 {
			ttl = 15 * time.Minute
		}
		request, err := s.approvalStore.Create(ctx, domain.ApprovalRequest{
			TenantID:  event.TenantID,
			Event:     event,
			Result:    result,
			Reason:    result.Reason,
			ExpiresAt: time.Now().UTC().Add(ttl),
		})
		if err != nil {
			return domain.EvaluationResult{}, fmt.Errorf("create approval request: %w", err)
		}
		result.ApprovalID = request.ID
	}
	return s.appendAudit(ctx, event, result)
}

func (s *Service) appendAudit(ctx context.Context, event domain.RuntimeEvent, result domain.EvaluationResult) (domain.EvaluationResult, error) {
	record, err := s.store.Append(ctx, domain.AuditRecord{
		Event:  event,
		Result: result,
	})
	if err != nil {
		return domain.EvaluationResult{}, fmt.Errorf("append audit record: %w", err)
	}
	result.AuditID = record.ID
	return result, nil
}

func (s *Service) evaluateApproval(ctx context.Context, event domain.RuntimeEvent) domain.EvaluationResult {
	if s.approvalStore == nil {
		return domain.EvaluationResult{
			Decision: domain.DecisionDeny,
			Reason:   "approval store is not configured",
		}
	}
	request, err := s.approvalStore.Get(ctx, event.TenantID, event.ApprovalID)
	if err != nil {
		return domain.EvaluationResult{
			Decision: domain.DecisionDeny,
			Reason:   "approval request not found",
		}
	}
	if request.Status != domain.ApprovalStatusApproved {
		return domain.EvaluationResult{
			Decision:   domain.DecisionDeny,
			Reason:     fmt.Sprintf("approval status is %s", request.Status),
			ApprovalID: request.ID,
		}
	}
	if !approvalEventMatches(request.Event, event) {
		return domain.EvaluationResult{
			Decision:   domain.DecisionDeny,
			Reason:     "approval request does not match runtime event",
			ApprovalID: request.ID,
		}
	}
	return domain.EvaluationResult{
		Decision:   domain.DecisionAllow,
		Reason:     "approved request matched runtime event",
		ApprovalID: request.ID,
	}
}

func approvalEventMatches(approved domain.RuntimeEvent, actual domain.RuntimeEvent) bool {
	return approved.TenantID == actual.TenantID &&
		approved.AgentID == actual.AgentID &&
		approved.UserID == actual.UserID &&
		approved.TaskID == actual.TaskID &&
		approved.EventType == actual.EventType &&
		approved.ToolName == actual.ToolName &&
		approved.Resource == actual.Resource &&
		approved.Action == actual.Action &&
		sameStringSet(approved.DataLabels, actual.DataLabels)
}

func sameStringSet(a []string, b []string) bool {
	left := append([]string(nil), a...)
	right := append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}
