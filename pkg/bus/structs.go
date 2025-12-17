package bus

type ApiRequest struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
}

type ApiResponse struct {
	Endpoint string            `json:"endpoint"`
	Params   map[string]string `json:"params"`
	Body     string            `json:"body"`
}
