package model

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SuccessResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}
