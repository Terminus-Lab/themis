package prechecks

import (
	"sync"

	"github.com/Terminus-Lab/themis/internal/models"
)

type StageRunner struct {
	Checkers []Checker
}

func NewStageRunner(checkers []Checker) *StageRunner {
	return &StageRunner{
		Checkers: checkers,
	}
}

func (r *StageRunner) Run(evaluationContext models.EvaluationContext) []models.StageResult {
	results := make(chan models.StageResult, len(r.Checkers))
	var wg sync.WaitGroup

	for _, checker := range r.Checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			results <- c.Check(evaluationContext)
		}(checker)
	}

	wg.Wait()
	close(results)

	var stageResults []models.StageResult
	for res := range results {
		stageResults = append(stageResults, res)
	}

	return stageResults
}
