package dto

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type CategoryDto struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	IconURL *string `json:"icon_url"`
}

type AdDto struct {
	ID       string `json:"id"`
	ImageURL string `json:"image_url"`
	Title    string `json:"title"`
	Deeplink string `json:"deeplink"`
}
