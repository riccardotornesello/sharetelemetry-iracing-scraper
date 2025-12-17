package bus

type ApiRequest struct {
	Endpoint string                 `json:"endpoint"`
	Params   map[string]interface{} `json:"params"`
}

type ApiResponse struct {
	Endpoint string                 `json:"endpoint"`
	Params   map[string]interface{} `json:"params"`
	Body     string                 `json:"body"`
}
