package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/HundredBai-hub/agent-secyrity/internal/audit"
	"github.com/HundredBai-hub/agent-secyrity/internal/domain"
	"github.com/HundredBai-hub/agent-secyrity/internal/policy"
)

type Service struct {
	engine *policy.Engine
	store  audit.Store
}

func NewService(engine *policy.Engine, store audit.Store) *Service {
	return &Service{engine: engine, store: store}
}

func (s *Service) Evaluate(ctx context.Context, event domain.RuntimeEvent) (domain.EvaluationResult, error) {
	if err := event.Validate(); err != nil {
		return domain.EvaluationResult{}, fmt.Errorf("%w: %v", domain.ErrInvalidRuntimeEvent, err)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	result := s.engine.Evaluate(event)
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
