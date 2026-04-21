package requests

type RequestInput struct {
	UserID  string      `json:"user_id"`
	Payload interface{} `json:"payload"`
}