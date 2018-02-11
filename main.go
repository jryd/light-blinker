package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/parnurzeal/gorequest"
)

// requestCounter is a global variable that we use to keep track
// of the number of blink requests we've made.
var requestCounter int

// thingspeakURL is a global variable that we use to store the baseURL
// that is used to make API requests to ThingSpeak.
var thingspeakURL string

// rateLimit is a global variable that we use to store the number of
// requests we are allowed to make.
// It is topped every 65 seconds (to allow for time variance on the host)
// with another 20 requests.
var rateLimit int

func main() {
	loadEnv()

	request := gorequest.New().SetDebug(false)

	// So the graph starts from 0
	updateThingSpeak(request, 0)

	// Set the initial request bucket to 20
	rateLimit = 20

	go topUpLimiter()

	for {
		csrfToken, csrfTokenFound := getCSRFToken(request)

		if csrfTokenFound {
			makeTheLightsBlinkTheRainbow(request, csrfToken)
		} else {
			updateThingSpeak(request, 0)
		}
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
func getCSRFToken(request *gorequest.SuperAgent) (string, bool) {
	res, _, _ := request.Get("http://blink.mattstauffer.com/").End()

	doc, err := goquery.NewDocumentFromResponse(res)

	if err != nil {
		log.Fatal(err)
	}

	csrfToken := ""
	csrfTokenFound := false

	doc.Find("meta").Each(func(index int, item *goquery.Selection) {
		if item.AttrOr("name", "") == "csrf-token" {
			csrfToken = item.AttrOr("content", "")
			csrfTokenFound = true
		}
	})

	return csrfToken, csrfTokenFound
}

// makeTheLightsBlinkTheRainbow fires 7 POST requests; one for each colour
// of the rainbow.
//
// It does this only 7 times, so that we can fetch a new CSRF token.
// This is done to prevent cases where the CSRF token may expire on us.
//
// If we do hit the rate limit for any reason, the requests will pause for
// a minute before continuing.
func makeTheLightsBlinkTheRainbow(request *gorequest.SuperAgent, csrfToken string) {
	for i := 0; i <= 7; i++ {

		colours := [7]string{"red", "orange", "yellow", "green", "blue", "purple", "pink"}

		for _, colour := range colours {

			if rateLimit > 0 {

				response, _, _ := request.Post("http://blink.mattstauffer.com/flash").
					Set("X-CSRF-TOKEN", csrfToken).
					Send(fmt.Sprintf(`{"color":"%v"}`, colour)).
					End()

				requestCounter++
				rateLimit--

				if response.StatusCode == 429 {
					// We've hit the rate limit, let's cool off before we blow the bulb ;)
					fmt.Println("Whoops we hit the rate limit - let's cool off for a bit")

					time.Sleep(60 * time.Second)
				}
			} else {
				time.Sleep(1 * time.Second)
			}

		}
	}

	updateThingSpeak(request, requestCounter)
}

// updateThingSpeak is responsible for updating our ThingSpeak graph so that
// we can keep track of how many requests we have made.
// This isn't critical to the running of the program, it's just cool to see
// the magnitude of blinks we're responsible for after this has run for a while :).
func updateThingSpeak(request *gorequest.SuperAgent, blinks int) {
	request.Get(fmt.Sprintf("%v%v", thingspeakURL, blinks)).End()
}

// topUpLimiter is responsible for topping up our rateLimit global variable.
// The app we are hitting allows 20 requests per minute - so we should respect
// that and make the most of it.
func topUpLimiter() {
	for range time.Tick(61 * time.Second) {
		rateLimit += 20
	}
}
