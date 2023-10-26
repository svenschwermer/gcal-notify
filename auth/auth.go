package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/svenschwermer/gcal-notify/browser"
	"github.com/svenschwermer/gcal-notify/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func configFromDisk() (*oauth2.Config, error) {
	clientSecretJSON, err := os.ReadFile(config.Cfg.ClientSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client secret file: %w", err)
	}
	config, err := google.ConfigFromJSON(clientSecretJSON, calendar.CalendarEventsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client secret file to config: %w", err)
	}
	return config, err
}

func GetTokenSourceFromWeb(ctx context.Context) (oauth2.TokenSource, error) {
	config, err := configFromDisk()
	if err != nil {
		return nil, err
	}
	lis, err := net.Listen("tcp4", "localhost:")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	config.RedirectURL = "http://" + lis.Addr().String()

	authCode := make(chan string)
	authError := make(chan error)
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			authError <- fmt.Errorf("failed to parse query: %w", err)
			rw.WriteHeader(http.StatusBadRequest)
		} else {
			codes, ok := query["code"]
			if ok {
				authCode <- codes[0]
				rw.Write([]byte("Authorized! You may close this windows now."))
			} else {
				authError <- errors.New("code query parameter missing")
				rw.WriteHeader(http.StatusBadRequest)
			}
		}
	})
	go http.Serve(lis, nil)

	authURL := config.AuthCodeURL("", oauth2.AccessTypeOffline)
	if !browser.Open(authURL) {
		fmt.Printf("Go to the following link in your browser:\n\n%v\n\n", authURL)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	select {
	case code := <-authCode:
		tok, err := config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve token from web: %w", err)
		}
		return config.TokenSource(ctx, tok), nil
	case err := <-authError:
		return nil, fmt.Errorf("oauth2 exchange failed: %w", err)
	case <-ctx.Done():
		return nil, errors.New("oauth2 exchange timed out")
	}
}

func GetTokenSourceFromDisk(ctx context.Context) (oauth2.TokenSource, error) {
	cfg, err := configFromDisk()
	if err != nil {
		return nil, err
	}
	tokenBytes, err := os.ReadFile(config.Cfg.TokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}
	tok := new(oauth2.Token)
	if err := json.Unmarshal(tokenBytes, tok); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	return cfg.TokenSource(ctx, tok), nil
}

func WriteTokenToDisk(ts oauth2.TokenSource, fatal bool) {
	if err := writeTokenToDisk(ts); err != nil {
		logger := log.Printf
		if fatal {
			logger = log.Fatalf
		}
		logger("Failed to write auth token to disk: %v", err)
	}
}

func writeTokenToDisk(ts oauth2.TokenSource) error {
	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("failed to get token from token source: %w", err)
	}
	tokenBytes, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	if err := os.MkdirAll(path.Dir(config.Cfg.TokenPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	return os.WriteFile(config.Cfg.TokenPath, tokenBytes, 0600)
}
