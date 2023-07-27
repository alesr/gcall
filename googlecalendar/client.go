package googlecalendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	tokenStash          string        = "/tmp/gcall-token"
	credentialsFilename string        = ".gcall"
	redirectURL         string        = "http://localhost:8080/auth"
	timeZone            string        = "Europe/Bucharest"
	defaultMeetingName  string        = "Instant meeting"
	authApprovalTimeout time.Duration = 30 * time.Second
)

type Client struct {
	logger          *zap.Logger
	httpClient      *http.Client
	codeChan        chan string
	calendarService *calendar.Service
}

// NewClient creates a new Client instance.
// TODO: inject the calendar service as a dependency.
func NewClient(logger *zap.Logger, codeCh chan string) (*Client, error) {
	client := Client{
		logger:   logger,
		codeChan: codeCh,
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home dir: %w", err)
	}
	credentialsPath := fmt.Sprintf("%s%s%s", homeDir, string(os.PathSeparator), credentialsFilename)

	credsB, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read credentials file: %w", err)
	}

	cfg, err := google.ConfigFromJSON(credsB, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("could not parse credentials file: %w", err)
	}

	tkn, err := client.getToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not get token: %w", err)
	}

	client.httpClient = cfg.Client(context.Background(), tkn)

	// TODO: replace deprecated constructor
	calendarService, err := calendar.New(client.httpClient)
	if err != nil {
		return nil, fmt.Errorf("could not create calendar service: %w", err)
	}

	client.calendarService = calendarService

	return &client, nil
}

// CreateInstantCall creates an instant call.
// It creates a new event in the primary calendar with the given name and duration.
func (c *Client) CreateInstantCall(meetingName string, duration time.Duration) (string, error) {
	event := &calendar.Event{
		Summary: meetingName,
		Start: &calendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
			TimeZone: timeZone,
		},
		End: &calendar.EventDateTime{
			DateTime: time.Now().Add(duration * time.Minute).Format(time.RFC3339),
			TimeZone: timeZone,
		},
		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: uuid.New().String(),
			},
		},
	}

	event, err := c.calendarService.Events.Insert("primary", event).ConferenceDataVersion(1).Do()
	if err != nil {
		return "", fmt.Errorf("could not create event: %w", err)
	}

	if event.ConferenceData.EntryPoints == nil {
		return "", fmt.Errorf("could not create event: no entry points")
	}

	for _, entryPoint := range event.ConferenceData.EntryPoints {
		if entryPoint.EntryPointType == "video" {
			return entryPoint.Uri, nil
		}
	}
	return "", errors.New("could not create event: no video entry point")
}

// getToken gets a token.
// If a token is stashed, it returns the stashed token.
// Otherwise, it gets a new token from the auth provider.
func (c *Client) getToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	stashedTkn, err := c.getStashedToken()
	if err != nil {
		log.Printf("could not get stashed token: %f", err)

		newTkn, err := c.getNewToken(cfg)
		if err != nil {
			return nil, fmt.Errorf("could not get new token: %w", err)
		}

		if err := c.stashToken(newTkn); err != nil {
			return nil, fmt.Errorf("could not stash token: %w", err)
		}
		return newTkn, nil
	}

	return stashedTkn, nil
}

// stashToken writes the given token to the token stash file.
func (c *Client) stashToken(tkn *oauth2.Token) error {
	tknB, err := json.Marshal(tkn)
	if err != nil {
		return fmt.Errorf("could not marshal token: %w", err)
	}

	if err := os.WriteFile(tokenStash, tknB, 0600); err != nil {
		return fmt.Errorf("could not write token file: %w", err)
	}

	return nil
}

// getStashedToken reads the token stash file and returns the token.
func (c *Client) getStashedToken() (*oauth2.Token, error) {
	tokenBytes, err := os.ReadFile(tokenStash)
	if err != nil {
		return nil, fmt.Errorf("could not read token file: %v", err)
	}

	if len(tokenBytes) == 0 {
		return nil, fmt.Errorf("token file is empty")
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("could not unmarshal token: %w", err)
	}

	return &token, nil
}

// getNewToken gets a new token from the auth provider.
// It awaits for the auth code coming from the callback server to be sent on the code channel.
func (c *Client) getNewToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	authURL := cfg.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("redirect_uri", redirectURL),
	)

	fmt.Printf("Visit the URL for the auth dialog: %s", authURL)

	select {
	case authCode := <-c.codeChan:
		// exchange the code for a token
		tkn, err := cfg.Exchange(
			context.Background(),
			authCode,
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("redirect_uri", redirectURL),
			oauth2.SetAuthURLParam("prompt", "consent"),
		)
		if err != nil {
			return nil, fmt.Errorf("could not exchange token: %w", err)
		}
		return tkn, nil
	case <-time.After(authApprovalTimeout):
		return nil, fmt.Errorf("timed out waiting for auth code")
	}
}
