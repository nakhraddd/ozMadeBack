package handlers

import (
	"net/http"
	"ozMadeBack/internal/dto"
	"ozMadeBack/internal/models"
	"ozMadeBack/internal/services"
	"ozMadeBack/internal/ui/admin"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	AdminService *services.AdminService
}

// Render renders a Templ component
func Render(c *gin.Context, status int, template templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	template.Render(c.Request.Context(), c.Writer)
}

// --- UI Handlers ---

func (h *AdminHandler) UIUsers(c *gin.Context) {
	users, err := h.AdminService.GetUsers()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	Render(c, http.StatusOK, admin.UserList(users))
}

func (h *AdminHandler) UIPendingProducts(c *gin.Context) {
	products, err := h.AdminService.GetPendingReviewProducts()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	Render(c, http.StatusOK, admin.ProductPending(products))
}

func (h *AdminHandler) UILogin(c *gin.Context) {
	Render(c, http.StatusOK, admin.Login())
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		AdminService: services.NewAdminService(db),
	}
}

// --- User Management Handlers ---

// GetUsers fetches all users
func (h *AdminHandler) GetUsers(c *gin.Context) {
	users, err := h.AdminService.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetUser fetches a single user by ID
func (h *AdminHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	user, err := h.AdminService.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var input models.User // Assuming models.User can be used for creation directly or create a specific DTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.AdminService.CreateUser(&input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to create user"})
		return
	}
	c.JSON(http.StatusCreated, input)
}

// UpdateUser updates an existing user
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	user, err := h.AdminService.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "User not found"})
		return
	}

	if err := c.ShouldBindJSON(&user); err != nil { // Bind updates to the fetched user object
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}
	user.ID = uint(userID) // Ensure the ID is correctly set from the URL parameter

	if err := h.AdminService.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to update user"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user by ID
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	if err := h.AdminService.DeleteUser(uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "User deleted successfully"})
}

// --- Product Review Handlers ---

// GetPendingReviewProducts fetches products that are pending admin review
func (h *AdminHandler) GetPendingReviewProducts(c *gin.Context) {
	products, err := h.AdminService.GetPendingReviewProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch pending review products"})
		return
	}
	c.JSON(http.StatusOK, products)
}

// ApproveProduct approves a product by ID
func (h *AdminHandler) ApproveProduct(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	if err := h.AdminService.ApproveProduct(uint(productID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to approve product"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Product approved successfully"})
}

// RejectProduct rejects a product by ID
func (h *AdminHandler) RejectProduct(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	if err := h.AdminService.RejectProduct(uint(productID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to reject product"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Product rejected successfully"})
}

// --- Seller Licenses Handlers ---

// GetSellerLicenses fetches licenses for a specific seller
func (h *AdminHandler) GetSellerLicenses(c *gin.Context) {
	sellerIDStr := c.Param("id")
	sellerID, err := strconv.ParseUint(sellerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid seller ID"})
		return
	}

	licenses, err := h.AdminService.GetSellerLicenses(uint(sellerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch seller licenses"})
		return
	}
	c.JSON(http.StatusOK, licenses)
}

// --- Report Review Handlers ---

// GetReports fetches all reports
func (h *AdminHandler) GetReports(c *gin.Context) {
	reports, err := h.AdminService.GetReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch reports"})
		return
	}
	c.JSON(http.StatusOK, reports)
}

// GetReport fetches a single report by ID
func (h *AdminHandler) GetReport(c *gin.Context) {
	reportIDStr := c.Param("id")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid report ID"})
		return
	}

	report, err := h.AdminService.GetReportByID(uint(reportID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to fetch report"})
		return
	}
	if report == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "Report not found"})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ResolveReport marks a report as resolved
func (h *AdminHandler) ResolveReport(c *gin.Context) {
	reportIDStr := c.Param("id")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid report ID"})
		return
	}

	if err := h.AdminService.ResolveReport(uint(reportID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to resolve report"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Report resolved successfully"})
}

// DismissReport marks a report as dismissed
func (h *AdminHandler) DismissReport(c *gin.Context) {
	reportIDStr := c.Param("id")
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid report ID"})
		return
	}

	if err := h.AdminService.DismissReport(uint(reportID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "Failed to dismiss report"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Report dismissed successfully"})
}
