package model

type BaseReponse struct {
	Code    int             `json:"code"`
	Message MessageResponse `json:"message"`
	Data    interface{}     `json:"data"`
}

type MessageResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}
