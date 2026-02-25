package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"backend-muslimverse/config"
	"backend-muslimverse/models"
)

var jwtSecret = []byte("rahasia_jwt_yang_sangat_rahasia_123")

type supabaseAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	User        struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"user"`
}

func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call Supabase REST API for signup
	url := config.SupabaseURL + "/auth/v1/signup"

	body := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal register: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var authResp supabaseAuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Gagal register"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Register berhasil! Silakan cek email untuk verifikasi.",
		"user": gin.H{
			"id":    authResp.User.ID,
			"email": authResp.User.Email,
		},
	})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call Supabase REST API for signin
	url := config.SupabaseURL + "/auth/v1/token?grant_type=password"

	body := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal login: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var authResp supabaseAuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	// Generate JWT token sendiri
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": authResp.User.ID,
		"email":   authResp.User.Email,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: tokenString,
		User: models.User{
			ID:        authResp.User.ID,
			Email:     authResp.User.Email,
			CreatedAt: authResp.User.CreatedAt,
		},
	})
}
