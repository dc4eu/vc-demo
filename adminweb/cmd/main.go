package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"strings"
	"vcweb1/pkg/configuration"
	"vcweb1/pkg/logger"
	"vcweb1/pkg/model"
	//Backup of some imports since the IDE sometimes removes them to fast
	//"bytes"
	//"encoding/json"
	//"github.com/gin-contrib/gzip"
	//"github.com/gin-contrib/sessions"
	//"github.com/gin-contrib/sessions/cookie"
	//"github.com/gin-gonic/gin"
	//"github.com/google/uuid"
	//"io"
	//"log"
	//"net/http"
	//"strings"
)

//TODO: Inför vettigare loggning (och ta bort log.Print...)
//TODO: inför timeout vid anrop
//TODO: inför rate-limit
//TODO: ...

const (
	//TODO: ta in apigwBaseUrl via config.yaml
	apigwBaseUrl    = "http://172.16.50.2:8080"
	apigwAPIBaseUrl = apigwBaseUrl + "/api/v1"
	//TODO: ta in sessionUserKey via config.yaml
	sessionUserKey = "user"
)

/* Secret for session cookie store (16-byte, 32-, ...) */
//TODO: ta sessionStoreSecret koden från config.yaml
var sessionStoreSecret = []byte("very-secret-code")

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
	router.Use(setupSessionMiddleware(sessionStoreSecret, 300, "/"))

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
		secureRouter.GET("/health", getHealthHandler)
		secureRouter.GET("/document/:document_id", getDocumentByIdHandler)
		secureRouter.GET("/devjsonobj", getDevJsonObjHandler)
		secureRouter.GET("/devjsonarray", getDevJsonArrayHandler)
		secureRouter.GET("/user", getUserHandler)
		secureRouter.GET("/loginstatus", getLoginStatusHandler)
	}

	//TODO: Inför https (TLS) stöd
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Unable to start gin engine:", err)
	}
}

func createMockHandler(cfg *model.Cfg, client *http.Client) gin.HandlerFunc {
	//closure
	return func(c *gin.Context) {
		url := apigwAPIBaseUrl + "/mock/next"
		doPostForDemoFlows(c, url, client)
	}
}

func doPostForDemoFlows(c *gin.Context, url string, client *http.Client) {
	//{
	//	document_type: <från dropdown>
	//	authentic_source: "SUNET", ta även in denna från GUI (sätt som statisk)
	//	authentic_source_person_id: <fritextfält i GUI>
	//}

	type Body struct {
		DocumentType            string `json:"document_type" binding:"required"`
		AuthenticSource         string `json:"authentic_source" binding:"required"`
		AuthenticSourcePersonId string `json:"authentic_source_person_id" binding:"required"`
	}

	var reqBody Body

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Print("reqBody: ", reqBody)

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshalling body"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"Error Unmarshal resp Body": err.Error()})
		return
	}

	log.Print("resp statuscode from vc: ", resp.StatusCode)

	c.JSON(resp.StatusCode, jsonResp)
}

func fetchFromPortalHandler(cfg *model.Cfg, client *http.Client) gin.HandlerFunc {
	//closure
	return func(c *gin.Context) {
		url := apigwAPIBaseUrl + "/portal"
		doPostForDemoFlows(c, url, client)
	}
}

func setupSessionMiddleware(secret []byte, maxAge int, path string) gin.HandlerFunc {
	// Configure session cookie store
	store := configureSessionStore(secret, maxAge, path)
	return sessions.Sessions("vcadminwebsession", store)
}

func configureSessionStore(secret []byte, maxAge int, path string) sessions.Store {
	store := cookie.NewStore(secret)
	store.Options(sessions.Options{
		Path:   path,
		MaxAge: maxAge, // 5 minuter i sekunder - javascript koden tar hänsyn till detta för att försöka gissa om användaren fortsatt är inloggad (om inloggad också vill säga)
		//Secure:   true,  //TODO: Aktivera för produktion för HTTPS
		HttpOnly: true,
	})
	return store
}

func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionUserKey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized/session expired"})
		return
	}

	if !isLogoutRoute(c) { // Don't touch the session (including cookie) during logout
		// Update MaxAge for the session and its cookie - extended time to expire with another 5 minutes from now
		session.Options(sessions.Options{
			MaxAge: 300, // 5 minuter
			Path:   "/",
			//Secure:   true,  //TODO: Aktivera för produktion för HTTPS
			HttpOnly: true,
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

func loginHandler(cfg *model.Cfg) func(c *gin.Context) {
	//closure
	return func(c *gin.Context) {
		session := sessions.Default(c)

		type LoginBody struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		var loginBody LoginBody
		if err := c.ShouldBindJSON(&loginBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if loginBody.Username != cfg.Web1.Username || loginBody.Password != cfg.Web1.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
			return
		}

		// TODO: ev. use a userID instead of username
		session.Set(sessionUserKey, loginBody.Username)
		if err := session.Save(); err != nil { //This is also where the cookie is created
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
	}
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionUserKey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	// Delete the session and cookie
	session.Delete(sessionUserKey)
	session.Options(sessions.Options{
		MaxAge: -1, // Expired
		Path:   "/",
		//Secure:   true,  //TODO: Aktivera för produktion för HTTPS
		HttpOnly: true,
	})
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove session (and cookie)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func getUserHandler(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(sessionUserKey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func getLoginStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}

func getHealthHandler(c *gin.Context) {
	//return func(c *gin.Context) {
	url := apigwBaseUrl + "/health"
	//log.Printf("URL: %s", url)

	//TODO: MS: vad är konceptet för att hantera/köra https client mot apigw?
	//TODO: lägga in timeout
	//TODO: på flera ställen: HTTP Client: You create a new HTTP client for every request. Instead of creating a new client for every HTTP request, use a single client for all requests can be beneficial so that TCP connections can be reused.
	client := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		//log.Printf("Error while preparing request to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
		return
	}

	resp, err := client.Do(req)
	//if resp != nil {
	//	log.Print("Respons header:", resp.Header)
	//}
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
		return
	}

	//log.Print("Response Body:", string(data))

	c.Data(resp.StatusCode, "application/json", data)
	//}
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

func getDocumentByIdHandler(c *gin.Context) {

	documentId := c.Param("document_id")

	if !isValidUUID(documentId) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "UUID expected or has wrong format"})
		return
	}

	url := apigwAPIBaseUrl + "/portal"
	//log.Printf("URL: %s", url)

	//TODO: MS: vad är konceptet för att hantera/köra https client mot apigw?
	//TODO: lägga in timeout
	client := http.Client{}

	jsonBody := map[string]string{
		//"authentic_source": "SUNET",
		//"document_id":      documentId,
		//"document_type":    "EHIC",
		"authentic_source":           "SUNET",
		"authentic_source_person_id": documentId,
		"validity_from":              "1970-01-01",
		"validity_to":                "1970-01-01",
	}

	/*
		/api/v1/portal

		"authentic_source"
		"authentic_source_person_id"
		"validity_from"
		"validity_to"
	*/

	// Serialize 'jsonBody' to JSON-format
	jsonData, err := json.Marshal(jsonBody)
	if err != nil {
		//log.Printf("Error marshalling jsonBody: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshalling jsonBody"})
		return
	}

	// Create new HTTP POST reguest with jsonData as Body
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		//log.Printf("Error while preparing request to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	//if resp != nil {
	//	log.Print("Respons header:", resp.Header)
	//}
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
		return
	}

	var jsonResp map[string]interface{}
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error Unmarshal resp Body": err.Error()})
		return
	}
	//log.Print("Response Body:", string(Body))

	c.JSON(resp.StatusCode, jsonResp)
}

/* TODO: remove before production */
func getDevJsonArrayHandler(c *gin.Context) {
	url := "http://jsonplaceholder.typicode.com/posts" //Just some random testserver on the internet that responds with a json array
	//log.Printf("URL: %s", url)

	client := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		//log.Printf("Error while preparing request to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
		return
	}

	resp, err := client.Do(req)
	//if resp != nil {
	//	log.Print("Respons header:", resp.Header)
	//}
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Printf("Error during reguest to url: %s %s", url, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
		return
	}

	//log.Print("Response Body:", string(data))

	c.Data(http.StatusOK, "application/json", data)
}

/*TODO: remove before production */
func getDevJsonObjHandler(c *gin.Context) {

	jsonData := gin.H{
		"message": "Dummy json object - hardcoded",
	}
	c.JSON(http.StatusOK, jsonData)
}
