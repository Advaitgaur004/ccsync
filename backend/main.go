package main

import (
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"ccsync_backend/controllers"
	"ccsync_backend/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	controllers.GlobalJobQueue = controllers.NewJobQueue()
	// OAuth2 client credentials
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SEC")
	redirectURL := os.Getenv("REDIRECT_URL_DEV")

	// OAuth2 configuration
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}

	// Create a session store
	sessionKey := []byte(os.Getenv("SESSION_KEY"))
	if len(sessionKey) == 0 {
		log.Fatal("SESSION_KEY environment variable is not set or empty")
	}
	store := sessions.NewCookieStore(sessionKey)
	gob.Register(map[string]interface{}{})

	app := controllers.App{Config: conf, SessionStore: store}
	mux := http.NewServeMux()

	//Rate limiter middleware that allows 50 requests per 30 seconds per IP
	rateLimitedHandler := func(handler http.HandlerFunc) http.Handler {
		return middleware.RateLimitMiddleware(handler, 30*time.Second, 50)
	}

	mux.Handle("/auth/oauth", rateLimitedHandler(app.OAuthHandler))
	mux.Handle("/auth/callback", rateLimitedHandler(app.OAuthCallbackHandler))
	mux.Handle("/api/user", rateLimitedHandler(app.UserInfoHandler))
	mux.Handle("/auth/logout", rateLimitedHandler(app.LogoutHandler))
	mux.Handle("/tasks", rateLimitedHandler(controllers.TasksHandler))
	mux.Handle("/add-task", rateLimitedHandler(controllers.AddTaskHandler))
	mux.Handle("/edit-task", rateLimitedHandler(controllers.EditTaskHandler))
	mux.Handle("/modify-task", rateLimitedHandler(controllers.ModifyTaskHandler))
	mux.Handle("/complete-task", rateLimitedHandler(controllers.CompleteTaskHandler))
	mux.Handle("/delete-task", rateLimitedHandler(controllers.DeleteTaskHandler))

	mux.HandleFunc("/ws", controllers.WebSocketHandler)

	go controllers.JobStatusManager()
	log.Println("Server started at :8000")
	if err := http.ListenAndServe(":8000", app.EnableCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
