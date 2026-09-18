package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/service"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauthConfig *oauth2.Config

func init() {
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func generateStateToken() string {
	return time.Now().Format("20060102150405")
}

// GoogleLogin godoc
// @Summary Redirect user to Google login page
// @Description Initiate Google OAuth2 login
// @Tags Auth
// @Produce json
// @Success 302 {string} string "Redirect to Google"
// @Router /auth/google/login [get]
func GoogleLogin(c *gin.Context) {
	state := generateStateToken()

	c.SetCookie("oauth_state", state, 300, "/", "jatidiri.app", true, true)

	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback godoc
// @Summary Handle Google OAuth2 callback
// @Description Handle redirect from Google and login/register user
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /auth/google/callback [get]
func GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	expectedState, err := c.Cookie("oauth_state")
	if err != nil || state != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("[OAuth] Token exchange failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange token"})
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch user info"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response from Google"})
		return
	}

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user info response"})
		return
	}

	accessToken, refreshToken, user, err := service.GoogleLoginCallback(
		userInfo.ID,
		userInfo.Email,
		userInfo.Name,
		userInfo.Picture,
		token.AccessToken,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("oauth_state", "", -1, "/", "jatidiri.app", true, true)

	c.JSON(http.StatusOK, gin.H{
		"message":       "google login success",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
	})
}
