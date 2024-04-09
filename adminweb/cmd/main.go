package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

//TODO: Inför någon enkel form av auth (login/logout inkl timeout) tills ev. en mer avancerad införs...(ex JWT)
//TODO: Inför vettigare loggning (och ta bort log.Print...)
//TODO: inför timeout vid anrop
//TODO: inför rate-limit
//TODO: ...

const userkey = "user"

func main() {
	//router := engine()
	//TODO: Inför https (TLS) stöd
	if err := engine().Run(":8080"); err != nil {
		log.Fatal("Unable to start:", err)
	}
}

func engine() *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	//TODO: router.Use(gin.MinifyHTML())
	//TODO: ??? router.Use(gin.Gzip())

	// Hemlighet för session cookie store (16-byte, 32-, ...)
	var secret = []byte("very-secret-code")

	// Konfigurera session cookie store
	store := cookie.NewStore(secret)
	store.Options(sessions.Options{
		Path:   "/",
		MaxAge: 300, // 5 minuter i sekunder - javascript koden tar hänsyn till detta för att försöka gissa om användaren fortsatt är inloggad (om inloggad också vill säga)
		//Secure:   true,  // Aktivera för produktion för HTTPS
		//HttpOnly: true,  // Förhindrar JavaScript-åtkomst
	})

	router.Use(sessions.Sessions("vcadminwebsession", store))

	router.LoadHTMLFiles("./assets/index.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Login and logout routes
	router.POST("/login", login)
	router.GET("/logout", logout)

	router.GET("/health", getHealthHandler())
	router.GET("/documents", getDocumentHandler())
	router.GET("/devjsonobj", getDevJsonObjHandler())
	router.GET("/devjsonarray", getDevJsonArrayHandler())

	// Private group, require authentication to access
	private := router.Group("/private")
	private.Use(AuthRequired)
	{
		private.GET("/me", me)
		private.GET("/status", status)
	}
	return router
}

// AuthRequired is a simple middleware to check the session.
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

// login is a handler that parses a form and checks for specific data.
func login(c *gin.Context) {
	session := sessions.Default(c)

	type LoginBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var loginBody LoginBody

	// Försök deserialisera förfrågans body till LoginBody-strukturen
	if err := c.BindJSON(&loginBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	username := loginBody.Username
	password := loginBody.Password

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	// Check for username and password match, usually from a database
	//TODO: ta in godkända username och password från config fil
	if username != "admin" || password != "secret123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Save the username in the session
	session.Set(userkey, username) // In real world usage you'd set this to the users ID
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
}

// logout is the handler called for the user to log out.
func logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// me is the handler that will return the user information stored in the
// session.
func me(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// status is the handler that will tell the user whether it is logged in or not.
func status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "You are logged in"})
}

func getHealthHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		url := "http://172.16.50.2:8080/health"
		log.Printf("URL: %s", url)

		//TODO: MS: vad är konceptet för att hantera/köra https client mot apigw?
		client := http.Client{}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error while preparing request to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
			return
		}

		resp, err := client.Do(req)
		if resp != nil {
			log.Print("Respons header:", resp.Header)
		}
		if err != nil {
			log.Printf("Error during reguest to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
			return
		}

		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error during reguest to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
			return
		}

		log.Print("Response body:", string(data))

		c.Data(http.StatusOK, "application/json", data)
	}
}

func getDocumentHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		jsonData := gin.H{
			"message": "TODO impl call to real /documents resource from webapp, now hardcoded json",
		}
		c.JSON(http.StatusOK, jsonData)
	}
}

/* TODO: remove before production */
func getDevJsonArrayHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		url := "http://jsonplaceholder.typicode.com/posts" //Just some random testserver on the internet that responds with a json array
		log.Printf("URL: %s", url)

		client := http.Client{}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error while preparing request to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error creating new http req": err.Error()})
			return
		}

		resp, err := client.Do(req)
		if resp != nil {
			log.Print("Respons header:", resp.Header)
		}
		if err != nil {
			log.Printf("Error during reguest to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error req": err.Error()})
			return
		}

		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error during reguest to url: %s %s", url, err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"Error read resp": err.Error()})
			return
		}

		log.Print("Response body:", string(data))

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
