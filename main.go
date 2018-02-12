package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// requestCounter is a global variable that we use to keep track
// of the number of blink requests we've made.
var requestCounter int

// thingspeakURL is a global variable that we use to store the baseURL
// that is used to make API requests to ThingSpeak.
var thingspeakURL string

var colours = [7]string{"red", "orange", "yellow", "green", "blue", "purple", "pink"}

var nextColour int

var csrfToken string

type rateLimitTransport struct {
	limiter *rate.Limiter
	xport   http.RoundTripper
}

var _ http.RoundTripper = &rateLimitTransport{}

func newRateLimitTransport(r float64, xport http.RoundTripper) http.RoundTripper {
	return &rateLimitTransport{
		limiter: rate.NewLimiter(rate.Limit(r), 1),
		xport:   xport,
	}
}

func (t *rateLimitTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.limiter.Wait(r.Context())
	return t.xport.RoundTrip(r)
}

var myClient = http.Client{
	// Use a rate-limiting transport which falls back to the default
	Transport: newRateLimitTransport(0.32, http.DefaultTransport),
}

func main() {
	loadEnv()

	// So the graph starts from 0
	updateThingSpeak(0)

	setCSRFToken()

	for {
		makeTheLightsBlinkTheRainbow()
	}
}

// loadEnv loads our .env file and sets up any global variables
// we might need to access.
//
// In this case, we are just loading the Thingspeak API key as
// we will need that to make API calls to update the graph.
func loadEnv() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	thingspeakURL = fmt.Sprintf("https://api.thingspeak.com/update?api_key=%v&field1=", os.Getenv("THINGSPEAK_API_KEY"))

	fmt.Println(".env loaded!")
}

// getCSRFToken visits the website and searches the HTML of the
// page for the CSRF token.
//
// This is then used to make subsequent calls to blink the lights.
func setCSRFToken() {
	response, err := http.Get("http://blink.mattstauffer.com/")

	doc, err := goquery.NewDocumentFromResponse(response)

	if err != nil {
		log.Fatal(err)
	}

	doc.Find("meta").Each(func(index int, item *goquery.Selection) {
		if item.AttrOr("name", "") == "csrf-token" {
			csrfToken = item.AttrOr("content", "")
		}
	})
}

// makeTheLightsBlinkTheRainbow fires 7 POST requests; one for each colour
// of the rainbow.
//
// It does this only 7 times, so that we can fetch a new CSRF token.
// This is done to prevent cases where the CSRF token may expire on us.
//
// If we do hit the rate limit for any reason, the requests will pause for
// a minute before continuing.
func makeTheLightsBlinkTheRainbow() {

	if requestCounter%49 == 0 {
		//get new CSRF token
		fmt.Println("Setting new CSRF token.")
		setCSRFToken()
	}

	body := []byte(fmt.Sprintf(`{"color":"%v"}`, colours[getNextColour()]))

	req, err := http.NewRequest(http.MethodPost, "http://blink.mattstauffer.com/flash", bytes.NewBuffer(body))

	req.Header.Set("X-CSRF-TOKEN", csrfToken)

	if err != nil {
		log.Fatal(err)
	}

	response, err := myClient.Do(req)

	fmt.Println("Request made")
	requestCounter++

	if response.StatusCode == 429 {
		// We've hit the rate limit, let's cool off before we blow the bulb ;)
		fmt.Println("Whoops we hit the rate limit - let's cool off for a bit")

		time.Sleep(60 * time.Second)
	}

	if requestCounter%7 == 0 {
		updateThingSpeak(requestCounter)
	}
}

// updateThingSpeak is responsible for updating our ThingSpeak graph so that
// we can keep track of how many requests we have made.
// This isn't critical to the running of the program, it's just cool to see
// the magnitude of blinks we're responsible for after this has run for a while :).
func updateThingSpeak(blinks int) {
	http.Get(fmt.Sprintf("%v%v", thingspeakURL, blinks))
}

// getNextColour is responsible for returning the index value of the next
// colour we need to send.
// It then updates the nextColour reference to the next index value to be
// used.
func getNextColour() int {
	colourForThisIteration := nextColour

	if nextColour == 6 {
		nextColour = 0
	} else {
		nextColour++
	}

	return colourForThisIteration
}
