package circuit

type ExtractionResult struct {
	Provider   string   `json:"provider"`
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
}

type CircuitSpec struct {
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
	Status     string   `json:"status"`
}
