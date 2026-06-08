package runtime

import (
	"context"
	"fmt"
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
	if err := event.Validate(); err != nil {
		return domain.EvaluationResult{}, fmt.Errorf("%w: %v", domain.ErrInvalidRuntimeEvent, err)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
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
