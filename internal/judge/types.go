package judge

type judgeResponse struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}
