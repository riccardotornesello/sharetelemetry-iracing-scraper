package bus

type ApiRequest struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
	Chunks   bool              `json:"chunks,omitempty"`
}

type ApiResponse struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
	Body     string            `json:"body"`
	Chunks   *string           `json:"chunks,omitempty"`
}
