package config

// Config represents the complete evaluation configuration
type Config struct {
	LLMJudge       LLMJudgeConfig                `yaml:"llm_judge"`
	EvaluationType string                        `yaml:"evaluation_type"`
	Judges         map[string]JudgeConfiguration `yaml:"judges"`
	Aggregation    AggregationConfig             `yaml:"aggregation"`
}

// LLMJudgeConfig contains the global evaluation prompt and annotation labels
type LLMJudgeConfig struct {
	Prompt           string            `yaml:"prompt"`
	AnnotationLabels []AnnotationLabel `yaml:"annotation_labels"`
}

// AnnotationLabel defines a score label for human annotation (used in correlation analysis)
type AnnotationLabel struct {
	Score       int    `yaml:"score"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

// AggregationConfig contains weights for aggregating precheck and judge scores
type AggregationConfig struct {
	PrecheckWeight float64 `yaml:"precheck_weight"`
	JudgeWeight    float64 `yaml:"judge_weight"`
}
