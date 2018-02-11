// Copyright 2018 James Bannister. All rights reserved.

/*
	This is an example of how to scrape and use a CSRF token in a
	rate limited application.

	The rate limiting is done based on a time ticker which fires
	the function to make our HTTP call.
*/
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

// colours is a global variable that stores a slice of colours we want to
// request. Currently stores the closest we can get to a rainbow.
var colours = [7]string{"red", "orange", "yellow", "green", "blue", "purple", "pink"}

// nextColour is a global variable that stores an integer representing the
// next index of the colours slice we want.
var nextColour int

// csrfToken is a global variable storing the current CSRF token we need
// to use in our requests.
var csrfToken string

func main() {
	loadEnv()

	request := gorequest.New().SetDebug(false)

	// So the graph starts from 0
	updateThingSpeak(request, 0)

	// Initially set our CSRF token
	setCSRFToken(request)

	// Set the rate at which calls should be made - time span / requests per time span
	rate := (61 * time.Second) / 20

	// Create the throttle ticker using the rate above
	throttle := time.Tick(rate)

	for {
		<-throttle  // rate limit our API calls
		go makeTheLightsBlinkTheRainbow(request, csrfToken)
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
func setCSRFToken(request *gorequest.SuperAgent) {
	res, _, _ := request.Get("http://blink.mattstauffer.com/").End()

	doc, err := goquery.NewDocumentFromResponse(res)

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
func makeTheLightsBlinkTheRainbow(request *gorequest.SuperAgent, csrfToken string) {

	if requestCounter % 49 == 0 {
		//get new CSRF token
		fmt.Println("Setting new CSRF token.")
		setCSRFToken(request)
	}

	response, _, _ := request.Post("http://blink.mattstauffer.com/flash").
		Set("X-CSRF-TOKEN", csrfToken).
		Send(fmt.Sprintf(`{"color":"%v"}`, colours[getNextColour()])).
		End()

	requestCounter++

	if response.StatusCode == 429 {
		fmt.Println("Whoops we hit the rate limit - let's cool off for a bit")

		time.Sleep(60 * time.Second)
	} else if response.StatusCode == 500 {
		fmt.Println("Whoops we got a 500 error - we probably have an invalid CSRF token!")
	}

	if requestCounter % 7 == 0 {
		updateThingSpeak(request, requestCounter)
	}
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

// updateThingSpeak is responsible for updating our ThingSpeak graph so that
// we can keep track of how many requests we have made.
// This isn't critical to the running of the program, it's just cool to see
// the magnitude of blinks we're responsible for after this has run for a while :).
func updateThingSpeak(request *gorequest.SuperAgent, blinks int) {
	request.Get(fmt.Sprintf("%v%v", thingspeakURL, blinks)).End()
}