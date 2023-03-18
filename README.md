# gcall
GCALL creates instant Google Meet meetings via the terminal to avoid the need for clicking multiple buttons in the Google UI

The code consists of two Go files: `callback.go` and `client.go`.

`callback.go` defines an HTTP server that listens on port 8080 for a GET request to `/auth`. When a request is received, it extracts the value of the code parameter from the query string and sends it to a channel. The server can be started and stopped using the `Start` and `Stop` methods of the Server struct.

`client.go` defines a Google Calendar API client that uses OAuth2 for authentication. When `NewClient` is called, it reads the client credentials from a JSON file, retrieves an OAuth2 token, and creates a `calendar.Service` object that can be used to interact with the Google Calendar API. The client exposes a method `CreateInstantCall` that creates an instant meeting on Google Meet and returns the URL for the video conference. The client also has methods for obtaining and storing the OAuth2 token. The `codeChan` parameter passed to `NewClient` is used to receive the authorization code sent by Google after the user grants permission for the application to access their Google Calendar data.
