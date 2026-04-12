package dto

type SendMessageInput struct {
	Content   string `json:"content"`
	MediaUrl  string `json:"media_url"`
	MediaType string `json:"media_type"`
}

type InitiateChatInput struct {
	ProductID uint   `json:"product_id" binding:"required"`
	Content   string `json:"content"`
}
