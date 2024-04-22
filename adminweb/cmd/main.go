package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"vcweb1/pkg/configuration"
	"vcweb1/pkg/logger"
	"vcweb1/pkg/model"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//TODO: add logging

const (
	/* session... constants is also used for the session cookie */
	sessionName                       = "vcadminwebsession" //if changed, the web (javascript) must also be updated with the new name
	sessionKey                        = "user"
	sessionInactivityTimeoutInSeconds = 3600 //one hour - also the value for the cookie
	sessionPath                       = "/"
	sessionHttpOnly                   = true
	sessionSecure                     = false //TODO: activate for https
	sessionSameSite                   = http.SameSiteStrictMode
)

func main() {
	ctx := context.Background()
	cfg, err := configuration.Parse(ctx, logger.NewSimple("Configuration"))
	if err != nil {
		panic(err)
	}

	router := gin.New()

	router.Use(gin.Logger())
	//TODO: router.Use(gin.MinifyHTML())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(setupSessionMiddleware(cfg))

	// Static route
	router.Static("/assets", "./assets")
	router.LoadHTMLFiles("./assets/index.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	httpClient := http.Client{}

	// Login route
	router.POST("/login", loginHandler(cfg))

	// Secure route group, require authentication to access
	secureRouter := router.Group("/secure")
	secureRouter.Use(authRequired)
	{
		secureRouter.POST("/mock", createMockHandler(cfg, &httpClient))
		secureRouter.POST("/portal", fetchFromPortalHandler(cfg, &httpClient))
		secureRouter.DELETE("/logout", logoutHandler)
		secureRouter.GET("/health", getHealthHandler(cfg, &httpClient))
		secureRouter.GET("/user", getUserHandler)
	}

	//TODO: add https (TLS) support
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Unable to start gin engine:", err)
	}
}

func createMockHandler(cfg *model.Cfg, client *http.Client) gin.HandlerFunc {
	//closure
	return func(c *gin.Context) {
		url := cfg.Web1.Services.MockAS.Addr + "/api/v1/mock/next"
		doPostForDemoFlows(c, url, client)
	}
}

func fetchFromPortalHandler(cfg *model.Cfg, client *http.Client) gin.HandlerFunc {
	//closure
	return func(c *gin.Context) {
		url := cfg.Web1.Services.APIGW.Addr + "/api/v1/portal"
		doPostForDemoFlows(c, url, client)
	}
}

type demoFlowRequestBody struct {
	DocumentType            string `json:"document_type" binding:"required"`
	AuthenticSource         string `json:"authentic_source" binding:"required"`
	AuthenticSourcePersonId string `json:"authentic_source_person_id" binding:"required"`
}

func doPostForDemoFlows(c *gin.Context, url string, client *http.Client) {
	var reqBody demoFlowRequestBody

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshalling body"})
		return
	}

	doPostRequest(c, url, client, err, reqBodyJSON)
}

func doPostRequest(c *gin.Context, url string, client *http.Client, err error, reqBodyJSON []byte) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
		return
	}

	var jsonResp map[string]interface{}
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error Unmarshal response to json": err.Error()})
		return
	}

	c.JSON(resp.StatusCode, jsonResp)
}

func setupSessionMiddleware(cfg *model.Cfg) gin.HandlerFunc {
	// Configure session cookie store
	store := configureSessionStore(cfg)
	return sessions.Sessions(sessionName, store)
}

func configureSessionStore(cfg *model.Cfg) sessions.Store {
	//The first parameter is used to encrypt and decrypt cookies.
	//The second parameter is used internally by cookie.Store to handle the encryption and decryption process
	store := cookie.NewStore([]byte(cfg.Web1.SessionCookieAuthenticationKey), []byte(cfg.Web1.SessionStoreEncryptionKey))
	store.Options(sessions.Options{
		Path:     sessionPath,
		MaxAge:   sessionInactivityTimeoutInSeconds,
		Secure:   sessionSecure,
		HttpOnly: sessionHttpOnly,
		SameSite: sessionSameSite,
	})
	return store
}

func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionKey)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized/session expired"})
		return
	}

	if !isLogoutRoute(c) { // Don't touch the session (including cookie) during logout
		// Update MaxAge for the session and its cookie - extended time to expire with another 5 minutes from now
		session.Options(sessions.Options{
			MaxAge:   sessionInactivityTimeoutInSeconds,
			Path:     sessionPath,
			Secure:   sessionSecure,
			HttpOnly: sessionHttpOnly,
			SameSite: sessionSameSite,
		})

		if err := session.Save(); err != nil {
			c.JSON(500, gin.H{"error": "Could not save session"})
			return
		}
	}

	c.Next()
}

func isLogoutRoute(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method
	return strings.HasSuffix(path, "/logout") && method == "DELETE"
}

type loginBody struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func loginHandler(cfg *model.Cfg) func(c *gin.Context) {
	//closure
	return func(c *gin.Context) {
		session := sessions.Default(c)

		var loginBody loginBody
		if err := c.ShouldBindJSON(&loginBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if loginBody.Username != cfg.Web1.Username || loginBody.Password != cfg.Web1.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
			return
		}

		// TODO: use a userID or UUID instead of username
		session.Set(sessionKey, loginBody.Username)
		if err := session.Save(); err != nil { //This is also where the cookie is created
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
	}
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionKey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	// Delete the session and cookie
	session.Delete(sessionKey)
	session.Options(sessions.Options{
		MaxAge:   -1, // Expired
		Path:     sessionPath,
		Secure:   sessionSecure,
		HttpOnly: sessionHttpOnly,
		SameSite: sessionSameSite,
	})
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove session (and cookie)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func getUserHandler(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionKey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func getHealthHandler(cfg *model.Cfg, client *http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := cfg.Web1.Services.APIGW.Addr + "/health"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
		}

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
		}

		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
		}

		c.Data(resp.StatusCode, "application/json", data)
	}

}

func isValidUUID(str string) bool {
	if str == "" {
		return false
	}
	if _, err := uuid.Parse(str); err != nil {
		return false
	}
	return true
}
