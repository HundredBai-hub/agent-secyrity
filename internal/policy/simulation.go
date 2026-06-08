package policy

import "github.com/HundredBai-hub/agent-secyrity/internal/domain"

const PolicySimulationSchemaV1 = "policy_simulation.v1"

type SimulationRequest struct {
	Event       domain.RuntimeEvent `json:"event"`
	PolicyPacks []domain.PolicyPack `json:"policy_packs"`
}

type SimulationResult struct {
	SchemaVersion string                  `json:"schema_version"`
	Result        domain.EvaluationResult `json:"result"`
}

func Simulate(request SimulationRequest) (SimulationResult, error) {
	event, err := request.Event.Normalize()
	if err != nil {
		return SimulationResult{}, &domain.RuntimeEventError{Err: err}
	}
	engine := NewEngineFromPacks(request.PolicyPacks)
	return SimulationResult{
		SchemaVersion: PolicySimulationSchemaV1,
		Result:        engine.Evaluate(event),
	}, nil
}
