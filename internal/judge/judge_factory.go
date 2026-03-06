package judge

import (
	"fmt"

	"github.com/rs/zerolog"
)

// JudgeFactory creates and manages judges by name for single-judge execution.
// It loads judges from YAML configuration.
type JudgeFactory struct {
	judges map[string]Judge
}

// NewJudgeFactory creates a factory from existing judges.
func NewJudgeFactory(judges []Judge, logger *zerolog.Logger) *JudgeFactory {
	// Create map by judge name for quick lookup
	judgesMap := make(map[string]Judge)
	for _, j := range judges {
		judgesMap[j.Name()] = j
	}

	logger.Info().Int("judge_count", len(judgesMap)).Msg("Judge factory initialized")

	return &JudgeFactory{
		judges: judgesMap,
	}
}

func (f *JudgeFactory) Get(judgeName string) (Judge, error) {
	judge, exist := f.judges[judgeName]
	if !exist {
		return nil, fmt.Errorf("judge not found")
	}

	return judge, nil
}
