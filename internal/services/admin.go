package services

import (
	"errors"
	"fmt" // Import fmt for error formatting
	"ozMadeBack/internal/models"

	"gorm.io/gorm"
)

// AdminService provides methods for admin operations
type AdminService struct {
	DB *gorm.DB
}

// NewAdminService creates a new AdminService
func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{DB: db}
}

// --- User Management ---

// GetUsers fetches all users
func (s *AdminService) GetUsers() ([]models.User, error) {
	var users []models.User
	if err := s.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserByID fetches a single user by ID
func (s *AdminService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // User not found
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (s *AdminService) CreateUser(user *models.User) error {
	return s.DB.Create(user).Error
}

// UpdateUser updates an existing user
func (s *AdminService) UpdateUser(user *models.User) error {
	return s.DB.Save(user).Error
}

// DeleteUser deletes a user by ID
func (s *AdminService) DeleteUser(userID uint) error {
	return s.DB.Delete(&models.User{}, userID).Error
}

// --- Product Review ---

// GetPendingReviewProducts fetches products that are pending admin review
func (s *AdminService) GetPendingReviewProducts() ([]models.Product, error) {
	var products []models.Product
	// Assuming a 'status' field in Product model, e.g., "pending", "approved", "rejected"
	if err := s.DB.Where("status = ?", "pending").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// ApproveProduct approves a product by ID
func (s *AdminService) ApproveProduct(productID uint) error {
	return s.DB.Model(&models.Product{}).Where("id = ?", productID).Update("status", "approved").Error
}

// RejectProduct rejects a product by ID
func (s *AdminService) RejectProduct(productID uint) error {
	return s.DB.Model(&models.Product{}).Where("id = ?", productID).Update("status", "rejected").Error
}

// --- Seller Licenses (moved from Product Review) ---

// GetSellerLicenses fetches licenses for a specific seller
func (s *AdminService) GetSellerLicenses(sellerID uint) ([]string, error) {
	var seller models.Seller
	if err := s.DB.First(&seller, sellerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("seller with ID %d not found", sellerID)
		}
		return nil, err
	}

	signedURLs := make([]string, 0, len(seller.Licenses))
	for _, licenseObjectName := range seller.Licenses {
		url, err := GenerateSignedURLForLicense(licenseObjectName)
		if err != nil {
			// Log the error but continue processing other licenses
			fmt.Printf("Error generating signed URL for seller license %s: %v\n", licenseObjectName, err)
			continue
		}
		signedURLs = append(signedURLs, url)
	}

	return signedURLs, nil
}

// --- Report Review ---

// GetReports fetches all reports
func (s *AdminService) GetReports() ([]models.Report, error) {
	var reports []models.Report
	// Assuming a 'Report' model exists
	if err := s.DB.Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

// GetReportByID fetches a single report by ID
func (s *AdminService) GetReportByID(reportID uint) (*models.Report, error) {
	var report models.Report
	if err := s.DB.First(&report, reportID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Report not found
		}
		return nil, err
	}
	return &report, nil
}

// ResolveReport marks a report as resolved
func (s *AdminService) ResolveReport(reportID uint) error {
	// Assuming a 'status' field in Report model, e.g., "pending", "resolved", "dismissed"
	return s.DB.Model(&models.Report{}).Where("id = ?", reportID).Update("status", "resolved").Error
}

// DismissReport marks a report as dismissed
func (s *AdminService) DismissReport(reportID uint) error {
	return s.DB.Model(&models.Report{}).Where("id = ?", reportID).Update("status", "dismissed").Error
}
