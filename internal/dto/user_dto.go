package dto

type UpdateUserProfileInput struct {
	Name       *string  `json:"name"`
	Address    *string  `json:"address"`
	AddressLat *float64 `json:"address_lat"`
	AddressLng *float64 `json:"address_lng"`
	PhotoUrl   *string  `json:"photo_url"`
	FCMToken   *string  `json:"fcm_token"`
}

type UpdateFCMTokenInput struct {
	FCMToken string `json:"fcm_token" binding:"required"`
}
