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

// Secret for session cookie store (16-byte, 32-, ...)
var sessionStoreSecret = []byte("very-secret-code")

const userkey = "user"

func main() {
	//TODO: Inför https (TLS) stöd
	if err := engine().Run(":8080"); err != nil {
		log.Fatal("Unable to start gin engine:", err)
		panic(err)
	}
}

func engine() *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	//TODO: router.Use(gin.MinifyHTML())
	//TODO: ??? router.Use(gin.Gzip())

	// Konfigurera session cookie store
	store := cookie.NewStore(sessionStoreSecret)
	//TODO: Se över sessionen och timeout, är det nedan som avses även för sessionen, dvs behöver den uppdateras vid varje authRequired precis som cookiens timeout?
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

	// Login route
	router.POST("/login", loginHandler)

	// Secure route group, require authentication to access
	secureRouter := router.Group("/secure")
	secureRouter.Use(authRequired)
	{
		secureRouter.GET("/logout", logoutHandler)
		secureRouter.GET("/health", getHealthHandler())
		secureRouter.GET("/documents", getDocumentHandler())
		secureRouter.GET("/devjsonobj", getDevJsonObjHandler())
		secureRouter.GET("/devjsonarray", getDevJsonArrayHandler())
		secureRouter.GET("/user", getUserHandler)
		secureRouter.GET("/loginstatus", getLoginStatusHandler)
	}

	return router
}

func authRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	//TODO: update the cookie with now+5 minutes for timeout samt behöver även sessionen uppdaterad för timeout på något sätt?

	// Continue down the chain to handler etc
	c.Next()
}

// loginHandler is a handler that parses a form and checks for specific data.
func loginHandler(c *gin.Context) {
	session := sessions.Default(c)

	type LoginBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var loginBody LoginBody
	if err := c.BindJSON(&loginBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if strings.Trim(loginBody.Username, " ") == "" || strings.Trim(loginBody.Password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	//TODO: load valid username(s) och password(s) från config fil (or db)
	if loginBody.Username != "admin" || loginBody.Password != "secret123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// TODO: use a userID instead of the username
	session.Set(userkey, loginBody.Username)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
}

// logoutHandler is the handler called for the user to log out.
func logoutHandler(c *gin.Context) {
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

// getUserHandler is the handler that will return the user information stored in the
// session.
func getUserHandler(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(userkey)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// getLoginStatusHandler is the handler that will tell the user whether it is logged in or not.
func getLoginStatusHandler(c *gin.Context) {
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
