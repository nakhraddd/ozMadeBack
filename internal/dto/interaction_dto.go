package dto

import "time"

type ReviewDto struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	UserName     string    `json:"user_name"`
	UserPhoto    string    `json:"user_photo"`
	Rating       float64   `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`
	Text         string    `json:"text"`
	ProductID    uint      `json:"product_id,omitempty"`
	ProductTitle string    `json:"product_title,omitempty"`
}

type ProductReviewsResponse struct {
	Summary struct {
		ProductID     uint    `json:"product_id"`
		AverageRating float64 `json:"average_rating"`
		RatingsCount  int64   `json:"ratings_count"`
		ReviewsCount  int64   `json:"reviews_count"`
	} `json:"summary"`
	Reviews []ReviewDto `json:"reviews"`
}

type SellerReviewsResponse struct {
	Header struct {
		SellerID      uint64  `json:"seller_id"`
		SellerName    string  `json:"seller_name"`
		ReviewsCount  int64   `json:"reviews_count"`
		AverageRating float64 `json:"average_rating"`
		RatingsCount  int     `json:"ratings_count"`
	} `json:"header"`
	Reviews []ReviewDto `json:"reviews"`
}

type PostCommentInput struct {
	Rating float64 `json:"rating" binding:"required"`
	Text   string  `json:"text"`
}

type ReportProductInput struct {
	Reason string `json:"reason" binding:"required"`
}
