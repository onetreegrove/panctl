package contract

type Meta struct {
	Provider  string `json:"provider,omitempty"`
	Profile   string `json:"profile"`
	RequestID string `json:"request_id"`
}

type Response struct {
	Status     string      `json:"status"`
	Data       any         `json:"data,omitempty"`
	Error      *Error      `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       Meta        `json:"meta"`
}

type Pagination struct {
	Page     int  `json:"page"`
	Limit    int  `json:"limit"`
	Total    int  `json:"total,omitempty"`
	HasMore  bool `json:"has_more"`
	NextPage int  `json:"next_page,omitempty"`
}
