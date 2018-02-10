# Light Blinker

Matt Stauffer made a cool little app that allows you to blink his smart light different colours.
You can check it out live on his site - http://blink.mattstauffer.com/.

This is a tongue-in-cheek app that automatically blinks the light in the colours of the rainbow.

It also served as a good opportunity/demonstration of how to scrape a CSRF token and use it for subsequent requests.

## Prerequisites

* Server or development environment with Go setup
* ThingSpeak Channel setup and the API Key for this channel

## Installation

1. `go get github.com/jryd/light-blinker`
2. `cd /path-to-go/src/github.com/jryd/light-blinker`
3. `go build`

## Running This

As this uses a `.env` file, you need to ensure that you run the executable from a directory that contains the `.env` file

### Running from project root

`./light-blinker` or however you run an executable on your system.

### Running from another programfolder

`cd /path-to-go/src/github.com/jryd/light-blinker && ./light-blinker`

### Copying executable to new folder and running from there

This is fine, just make sure a `.env` file exists in the new directory for the executable to use.