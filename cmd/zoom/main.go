package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"

	zoom "github.com/benbalter/zoom-go"
	"github.com/benbalter/zoom-go/config"
)

func printSetupInstructions() {
	fmt.Print(`In order to use Zoom Launcher, you need to create an OAuth app and authorize it to access your calendar. 
You can do it in four, not-so-easy steps:

1. Create a new project
	1. Go to https://console.developers.google.com
	2. Switch to your work account if need be (top right)
	3. Create a new project dropdown, top left next to your domain
2. Grant the project Calendar API access
	1. Click "Enable API"
	2. Type "Calendar" in the search box
	3. Click "Calendar API"
	4. Click "Enable"
3. Grab your credentials
	1. Click "Credentials" on the left side
	2. Create a new OAuth credential with type "other"
	3. Download the credential to ~/.config/google/client_secrets.json (icon, right side)
4. Run 'zoom -import=Downloads/client_secrets.json' and follow the instructions to authorize the app.
`)
}

func importGoogleClientConfig(provider config.Provider, filename string) error {
	conf, err := config.ReadGoogleClientConfigFromFile(filename)
	if err != nil {
		return err
	}

	return provider.StoreGoogleClientConfig(conf)
}

func authorizeAccount(provider config.Provider) error {
	authURL, err := zoom.GoogleCalendarAuthorizationURL(provider)
	if err != nil {
		return err
	}

	fmt.Println("Your browser is about to open. When it does, please authorize the application when prompted and paste the token it gives you below.")
	time.Sleep(5 * time.Second)
	open.Run(authURL)

	fmt.Print("Authorization token: ")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return errors.WithStack(err)
	}
	return zoom.HandleGoogleCalendarAuthorization(provider, authCode)
}

func main() {
	provider, err := config.NewFileProvider()
	if err != nil {
		fmt.Printf("unable to create file configuration provider: %+v\n", err)
		os.Exit(1)
	}

	importCredential := flag.String("import", "", "Full path to your downloaded Google OAuth2 client_secret JSON file")
	flag.Parse()

	if importCredential != nil && *importCredential != "" {
		fmt.Printf("Importing credentials from %q...\n", *importCredential)
		if err := importGoogleClientConfig(provider, *importCredential); err != nil {
			fmt.Printf("error importing credentials: %+v\n", err)
		}
	}

	if !provider.GoogleClientConfigExists() {
		printSetupInstructions()
		os.Exit(1)
	}

	if !provider.GoogleTokenExists() {
		if err := authorizeAccount(provider); err != nil {
			fmt.Printf("error authorizing: %+v\n", err)
			os.Exit(1)
		}
		fmt.Println("Stored credentials.")
	}

	calendar, err := zoom.NewGoogleCalendarService(provider)
	if err != nil {
		fmt.Printf("error creating google calendar client: %+v\n", err)
		os.Exit(1)
	}

	meeting, err := zoom.NextEvent(calendar)
	if err != nil {
		fmt.Printf("error fetching next meeting: %+v\n", err)
		os.Exit(1)
	}

	if meeting == nil {
		fmt.Println("No upcoming events found.")
		return
	}

	fmt.Printf("Your next meeting is %q, organized by %s.\n", meeting.Summary, meeting.Organizer.DisplayName)

	startTime, err := zoom.MeetingStartTime(meeting)
	if startTime.Sub(time.Now()) < 0 {
		fmt.Printf("It started %s.\n", zoom.HumanizedStartTime(meeting))
	} else {
		fmt.Printf("It starts %s.\n", zoom.HumanizedStartTime(meeting))
	}

	fmt.Printf("Calendar event URL: %s\n\n", meeting.HtmlLink)

	url, ok := zoom.MeetingURLFromEvent(meeting)
	if !ok {
		fmt.Println("No Zoom URL found in the meeting.")
		os.Exit(1)
	}

	if zoom.IsMeetingSoon(meeting) {
		fmt.Printf("Opening %s...\n", url)
		open.Run(url.String())
	} else {
		fmt.Printf("Zoom URL: %s\n", url)
	}
}
