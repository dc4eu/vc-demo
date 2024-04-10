package main

import (
	"bytes"
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
	apigwBaseUrl    = "http://172.16.50.2:8080"
	apigwAPIBaseUrl = apigwBaseUrl + "/api/v1"
	sessionUserKey  = "user"
)

/* Secret for session cookie store (16-byte, 32-, ...) */
//TODO: ta in från konfiguration
var sessionStoreSecret = []byte("very-secret-code")

func main() {
	router := gin.New()

	router.Use(gin.Logger())
	//TODO: router.Use(gin.MinifyHTML())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(setupSessionMiddleware(sessionStoreSecret, 300, "/"))

	// Static route
	router.LoadHTMLFiles("./assets/index.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Login route
	router.POST("/login", loginHandler)

	// Secure route group, require authentication to access
	secureRouter := router.Group("/secure")
	secureRouter.Use(authRequired)
	{
		secureRouter.DELETE("/logout", logoutHandler)
		secureRouter.GET("/health", getHealthHandler())
		secureRouter.GET("/document/:document_id", getDocumentByIdHandler())
		secureRouter.GET("/devjsonobj", getDevJsonObjHandler())
		secureRouter.GET("/devjsonarray", getDevJsonArrayHandler())
		secureRouter.GET("/user", getUserHandler)
		secureRouter.GET("/loginstatus", getLoginStatusHandler)
	}

	//TODO: Inför https (TLS) stöd
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Unable to start gin engine:", err)
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

func loginHandler(c *gin.Context) {
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

	//TODO: load valid username(s) och password(s) från config fil (or db)
	if loginBody.Username != "admin" || loginBody.Password != "secret123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// TODO: use a userID instead of the username
	session.Set(sessionUserKey, loginBody.Username)
	if err := session.Save(); err != nil { //This is also where the cookie is created
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
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

func getHealthHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		url := apigwBaseUrl + "/health"
		//log.Printf("URL: %s", url)

		//TODO: MS: vad är konceptet för att hantera/köra https client mot apigw?
		//TODO: lägga in timeout
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

		//log.Print("Response body:", string(data))

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

func getDocumentByIdHandler() func(c *gin.Context) {
	return func(c *gin.Context) {

		documentId := c.Param("document_id")

		if !isValidUUID(documentId) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "UUID expected or has wrong format"})
			return
		}

		url := apigwAPIBaseUrl + "/document"
		//log.Printf("URL: %s", url)

		//TODO: MS: vad är konceptet för att hantera/köra https client mot apigw?
		//TODO: lägga in timeout
		client := http.Client{}

		jsonBody := map[string]string{
			//TODO: magnus, vad krävs för indata?
			"documentid": documentId,
		}

		// Serialize 'jsonBody' to JSON-format
		jsonData, err := json.Marshal(jsonBody)
		if err != nil {
			//log.Printf("Error marshalling jsonBody: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshalling jsonBody"})
			return
		}

		// Create new HTTP POST reguest with jsonData as body
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			//log.Printf("Error while preparing request to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
			return
		}
		req.Header.Set("Content-Type", "application/json")

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

		//log.Print("Response body:", string(body))

		c.Data(resp.StatusCode, "application/json", body)
	}
}

/* TODO: remove before production */
func getDevJsonArrayHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
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

		//log.Print("Response body:", string(data))

		c.Data(http.StatusOK, "application/json", data)
	}
}

/*TODO: remove before production */
func getDevJsonObjHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		jsonData := gin.H{
			"message": "Dummy json object - hardcoded",
		}
		c.JSON(http.StatusOK, jsonData)
	}
}
