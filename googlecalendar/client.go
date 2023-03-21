package googlecalendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	tokenStash          string        = "/tmp/gcall-token"
	credentialsPath     string        = "gcall_credentials.json"
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

func NewClient(logger *zap.Logger, codeCh chan string) (*Client, error) {
	client := Client{
		logger:   logger,
		codeChan: codeCh,
	}

	credsB, err := ioutil.ReadFile(credentialsPath)
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

	calendarService, err := calendar.New(client.httpClient)
	if err != nil {
		return nil, fmt.Errorf("could not create calendar service: %w", err)
	}

	client.calendarService = calendarService

	return &client, nil
}

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

func (c *Client) stashToken(tkn *oauth2.Token) error {
	tknB, err := json.Marshal(tkn)
	if err != nil {
		return fmt.Errorf("could not marshal token: %w", err)
	}

	if err := ioutil.WriteFile(tokenStash, tknB, 0600); err != nil {
		return fmt.Errorf("could not write token file: %w", err)
	}
	return nil
}

func (c *Client) getStashedToken() (*oauth2.Token, error) {
	tknB, err := ioutil.ReadFile(tokenStash)
	if err != nil {
		return nil, fmt.Errorf("could not read token file: %v", err)
	}

	if len(tknB) == 0 {
		return nil, fmt.Errorf("token file is empty")
	}

	var tkn oauth2.Token
	if err := json.Unmarshal(tknB, &tkn); err != nil {
		return nil, fmt.Errorf("could not unmarshal token: %w", err)
	}
	return &tkn, nil
}

func (c *Client) getNewToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	authURL := cfg.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("redirect_uri", redirectURL),
	)

	fmt.Printf("Visit the URL for the auth dialog: %s", authURL)

	select {
	case authCode := <-c.codeChan:
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
