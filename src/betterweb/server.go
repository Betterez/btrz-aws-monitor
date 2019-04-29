package betterweb

import (
	"betterauth"
	"btrzaws"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

// InstanceMessage - instance availability message
type InstanceMessage struct {
	Instance        *btrzaws.BetterezInstance
	Time            time.Time
	UnhealthyChecks int
	RestartAttempts int
	HardRestarts    int
}

// ClientResponse - response to the client
type ClientResponse struct {
	TimeStamp time.Time
	Instances []*btrzaws.BetterezInstance
	Messages  []InstanceMessage
	Version   string
}

// HealthCheckServer - server to healthcheck instances
type HealthCheckServer struct {
	serverMux     *http.ServeMux
	serverPort    int
	ServerVersion string
	awsSession    *session.Session
	serverStatus  string
	usersTokens   map[string]int
	authenticator betterauth.Authenticator
}

// CreateHealthCheckServer - create the server
func CreateHealthCheckServer() (*HealthCheckServer, error) {
	result := &HealthCheckServer{
		serverPort:    3000,
		ServerVersion: "0.0.0.2",
		serverStatus:  "Idle",
		usersTokens:   make(map[string]int),
	}
	authenticator, err := betterauth.GetSQLiteAuthenticator("secrets/users.db")
	if err != nil {
		return nil, err
	}
	result.authenticator = authenticator
	return result, nil
}

// GetServerStatus - get current status
func (server *HealthCheckServer) GetServerStatus(awsSession *session.Session) string {
	return server.serverStatus
}

// SetSession - sets the aws session
func (server *HealthCheckServer) SetSession(awsSession *session.Session) {
	server.awsSession = awsSession
}

// SetListeningPort - Set the Listening Port
func (server *HealthCheckServer) SetListeningPort(port int) {
	server.serverPort = port
}

func (server *HealthCheckServer) insertUserWithLevel(level int) string {
	token := betterauth.RandStringRunes(40)
	server.usersTokens[token] = level
	return token
}

// Start -starts the server
func (server *HealthCheckServer) Start() error {
	if server.awsSession == nil {
		return errors.New("No aws session")
	}
	clientResponse := &ClientResponse{Instances: []*btrzaws.BetterezInstance{}, Version: "1.0.0.3"}
	go checkInstances(server.awsSession, clientResponse)
	server.serverMux = http.NewServeMux()
	server.serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server version", server.ServerVersion)
		fmt.Fprint(w, "Working!")
	})
	server.serverMux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server version", server.ServerVersion)
		fmt.Fprint(w, "ok")
	})

	server.serverMux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error  - %v", err), http.StatusBadRequest)
			return
		}
		userLevel, err := server.authenticator.GetUserLevel(r.FormValue("username"), r.FormValue("password"))
		if err != nil {
			http.Error(w, fmt.Sprintf("server error %v", err), http.StatusForbidden)
			return
		}
		if userLevel == 0 {
			http.Error(w, "User not found", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/json")
		token := server.insertUserWithLevel(userLevel)
		fmt.Fprintf(w, `{"user_level":%d,"auth_code":"%s","username":"%s","lang_code":"北京青年报记者昨"}`, userLevel, token, r.FormValue("username"))
		return
	})

	server.serverMux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		userAuth, err := server.getUserCreds(r)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if userAuth < 1 {
			http.Error(w, "Not authenticated", http.StatusForbidden)
			return
		}
		encoder := json.NewEncoder(w)
		w.Header().Set("Content-Type", "text/json")
		encoder.Encode(clientResponse)
	})
	server.serverStatus = "running"
	http.ListenAndServe(fmt.Sprintf(":%d", server.serverPort), server.serverMux)
	return nil
}

func (server *HealthCheckServer) getUserCreds(r *http.Request) (int, error) {
	err := r.ParseForm()
	if err != nil {
		return 0, err
	}
	userToken := r.FormValue("token")
	if userToken == "" {
		return 0, nil
	}
	value, found := server.usersTokens[userToken]
	if !found {
		return 0, nil
	}
	return value, nil
}
