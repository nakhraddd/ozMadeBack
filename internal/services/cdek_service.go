package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ozMadeBack/internal/dto"

	"github.com/google/uuid" // Import for generating unique IDs
)

const (
	cdekBaseURL = "https://api.edu.cdek.ru/v2" // CDEK Test API base URL
)

// CDEKService handles interactions with the CDEK API
type CDEKService struct {
	Account   string
	SecureKey string
	Client    *http.Client
	// TokenCache string // In a real app, you'd cache the token and refresh it
}

// NewCDEKService initializes a new CDEKService
func NewCDEKService() *CDEKService {
	return &CDEKService{
		Account:   os.Getenv("CDEK_ACCOUNT"),    // Load from environment variables
		SecureKey: os.Getenv("CDEK_SECURE_KEY"), // Load from environment variables
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// CDEKLocation represents a location returned by CDEK API
type CDEKLocation struct {
	Code        int     `json:"code"`
	CityUUID    string  `json:"city_uuid"`
	City        string  `json:"city"`
	CountryUUID string  `json:"country_uuid"`
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	Kladr       string  `json:"kladr"`
	Fias        string  `json:"fias"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"time_zone"`
}

// CDEKLocationSearchResponse is the structure for the /location/cities API response
type CDEKLocationSearchResponse []CDEKLocation

// CDEKCalculateRequest represents the request body for CDEK calculation API
type CDEKCalculateRequest struct {
	Type         int                  `json:"type"`        // 1 for door-door, 2 for door-warehouse, etc.
	TariffCode   int                  `json:"tariff_code"` // Specific tariff code
	FromLocation CDEKLocationRequest  `json:"from_location"`
	ToLocation   CDEKLocationRequest  `json:"to_location"`
	Packages     []CDEKPackageRequest `json:"packages"`
}

// CDEKLocationRequest is a simplified location for calculation
type CDEKLocationRequest struct {
	Code        int      `json:"code,omitempty"`
	FiasGuid    string   `json:"fias_guid,omitempty"`
	KladrCode   string   `json:"kladr_code,omitempty"`
	PostalCode  string   `json:"postal_code,omitempty"`
	CountryCode string   `json:"country_code,omitempty"`
	City        string   `json:"city,omitempty"`
	Address     string   `json:"address,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`  // Changed to pointer
	Longitude   *float64 `json:"longitude,omitempty"` // Changed to pointer
}

// CDEKPackageRequest represents a package in CDEK calculation API
type CDEKPackageRequest struct {
	Number string `json:"number"` // Required: Unique package number
	Weight int    `json:"weight"` // in grams
	Length int    `json:"length"` // in cm
	Width  int    `json:"width"`  // in cm
	Height int    `json:"height"` // in cm
}

// CDEKCalculateResponse represents the response from CDEK calculation API
type CDEKCalculateResponse struct {
	TariffCodes []struct {
		TariffCode   int     `json:"tariff_code"`
		TariffName   string  `json:"tariff_name"`
		DeliveryMode int     `json:"delivery_mode"`
		DeliverySum  float64 `json:"delivery_sum"`
		PeriodMin    int     `json:"period_min"`
		PeriodMax    int     `json:"period_max"`
		CalendarMin  int     `json:"calendar_min"`
		CalendarMax  int     `json:"calendar_max"`
		Currency     string  `json:"currency"`
		Services     []struct {
			Code string  `json:"code"`
			Sum  float64 `json:"sum"`
		} `json:"services"`
	} `json:"tariff_codes"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// CDEKAuthResponse represents the response from CDEK OAuth API
type CDEKAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	UUID        string `json:"uuid"`
}

// GetCDEKToken retrieves an authentication token from CDEK API
func (s *CDEKService) GetCDEKToken() (string, error) {
	if s.Account == "" || s.SecureKey == "" {
		return "", fmt.Errorf("CDEK_ACCOUNT or CDEK_SECURE_KEY environment variables are not set")
	}

	form := url.Values{}
	form.Add("client_id", s.Account)
	form.Add("client_secret", s.SecureKey)
	form.Add("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", cdekBaseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create CDEK auth request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response // Explicitly declare resp
	resp, err = s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make CDEK auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("CDEK auth API returned non-OK status: %d, body: %s", resp.StatusCode, respBody)
	}

	var authResp CDEKAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode CDEK auth response: %w", err)
	}

	if authResp.AccessToken == "" {
		return "", fmt.Errorf("CDEK auth response did not contain an access token")
	}

	return authResp.AccessToken, nil
}

// FindCDEKLocationCode finds the CDEK location code by city name
func (s *CDEKService) FindCDEKLocationCode(city string) (*CDEKLocation, error) {
	token, err := s.GetCDEKToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get CDEK auth token for location search: %w", err)
	}

	req, err := http.NewRequest("GET", cdekBaseURL+"/location/cities", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDEK location request: %w", err)
	}
	q := req.URL.Query()
	q.Add("city", city)
	q.Add("country_code", "KZ") // Assuming Kazakhstan for now
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+token)

	var resp *http.Response // Explicitly declare resp
	resp, err = s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make CDEK location request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CDEK location API returned non-OK status: %d, body: %s", resp.StatusCode, respBody)
	}

	var locations CDEKLocationSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, fmt.Errorf("failed to decode CDEK location response: %w", err)
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("CDEK location not found for city: %s", city)
	}

	// Return the first matching location. In a real scenario, you might need more sophisticated logic
	// to pick the most accurate location (e.g., by checking region, coordinates if provided).
	return &locations[0], nil
}

// CalculateIntercityFare calculates the shipping cost using CDEK API
func (s *CDEKService) CalculateIntercityFare(reqData *dto.IntercityEstimateRequest) (*dto.IntercityEstimateResponse, error) {
	// 1. Validate input (already done in handler, but good to have here too for robustness)
	if reqData.FromAddress.City == reqData.ToAddress.City {
		return nil, fmt.Errorf("origin and destination cities cannot be the same for intercity delivery")
	}
	if reqData.Package.WeightGrams <= 0 || reqData.Package.HeightCm <= 0 || reqData.Package.WidthCm <= 0 || reqData.Package.DepthCm <= 0 {
		return nil, fmt.Errorf("package weight and dimensions must be positive")
	}
	if reqData.FromAddress.FullAddress == "" || reqData.FromAddress.City == "" || reqData.ToAddress.FullAddress == "" || reqData.ToAddress.City == "" {
		return nil, fmt.Errorf("full address and city are required for both origin and destination")
	}

	// 2. Get CDEK location codes for fromAddress and toAddress
	fromCDEKLocation, err := s.FindCDEKLocationCode(reqData.FromAddress.City)
	if err != nil {
		return nil, fmt.Errorf("failed to find CDEK location for origin city '%s': %w", reqData.FromAddress.City, err)
	}
	toCDEKLocation, err := s.FindCDEKLocationCode(reqData.ToAddress.City)
	if err != nil {
		return nil, fmt.Errorf("failed to find CDEK location for destination city '%s': %w", reqData.ToAddress.City, err)
	}

	// 3. Construct CDEK API request payload
	fromLocationRequest := CDEKLocationRequest{
		Code:    fromCDEKLocation.Code,
		Address: reqData.FromAddress.FullAddress,
	}
	if reqData.FromAddress.Latitude != nil {
		fromLocationRequest.Latitude = reqData.FromAddress.Latitude
	}
	if reqData.FromAddress.Longitude != nil {
		fromLocationRequest.Longitude = reqData.FromAddress.Longitude
	}

	toLocationRequest := CDEKLocationRequest{
		Code:    toCDEKLocation.Code,
		Address: reqData.ToAddress.FullAddress,
	}
	if reqData.ToAddress.Latitude != nil {
		toLocationRequest.Latitude = reqData.ToAddress.Latitude
	}
	if reqData.ToAddress.Longitude != nil {
		toLocationRequest.Longitude = reqData.ToAddress.Longitude
	}

	cdekReq := CDEKCalculateRequest{
		Type:         1,   // Door-to-door delivery
		TariffCode:   136, // Example: specific tariff code for express delivery (adjust as needed)
		FromLocation: fromLocationRequest,
		ToLocation:   toLocationRequest,
		Packages: []CDEKPackageRequest{
			{
				Number: uuid.New().String(), // Generate a unique number for the package
				Weight: reqData.Package.WeightGrams,
				Length: reqData.Package.DepthCm, // Assuming depth is length for CDEK
				Width:  reqData.Package.WidthCm,
				Height: reqData.Package.HeightCm,
			},
		},
	}

	// 4. Get CDEK Auth Token
	token, err := s.GetCDEKToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get CDEK auth token: %w", err)
	}

	// 5. Make HTTP request to CDEK calculation API
	reqBody, err := json.Marshal(cdekReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CDEK request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", cdekBaseURL+"/calculator/tarifflist", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create CDEK HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	var httpResp *http.Response // Explicitly declare httpResp
	httpResp, err = s.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make CDEK API request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("CDEK API returned non-OK status: %d, body: %s", httpResp.StatusCode, respBody)
	}

	var cdekResp CDEKCalculateResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&cdekResp); err != nil {
		return nil, fmt.Errorf("failed to decode CDEK response: %w", err)
	}

	if len(cdekResp.Errors) > 0 {
		// Combine all errors into a single string
		errorMessages := make([]string, len(cdekResp.Errors))
		for i, e := range cdekResp.Errors {
			errorMessages[i] = fmt.Sprintf("Code: %s, Message: %s", e.Code, e.Message)
		}
		return nil, fmt.Errorf("CDEK API errors: %s", strings.Join(errorMessages, "; "))
	}
	if len(cdekResp.TariffCodes) == 0 {
		return nil, fmt.Errorf("no tariff codes found for the given parameters")
	}

	// 6. Process CDEK response and return the best estimate
	// For simplicity, taking the first tariff code. In a real app, you might choose based on price, speed, etc.
	bestTariff := cdekResp.TariffCodes[0]

	// Calculate estimated dates
	today := time.Now()
	estimatedDateFrom := today.AddDate(0, 0, bestTariff.PeriodMin).Format("2006-01-02")
	estimatedDateTo := today.AddDate(0, 0, bestTariff.PeriodMax).Format("2006-01-02")

	return &dto.IntercityEstimateResponse{
		Provider:          "CDEK",
		Price:             bestTariff.DeliverySum,
		Currency:          bestTariff.Currency,
		MinDays:           bestTariff.PeriodMin,
		MaxDays:           bestTariff.PeriodMax,
		EstimatedDateFrom: estimatedDateFrom,
		EstimatedDateTo:   estimatedDateTo,
	}, nil
}
