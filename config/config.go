// Config loads configuration.
package config

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

const Version = "1.1"

// GetInt loads the environment variable varName, converts it to an integer,
// and returns that integer or an error.
func GetInt(varName string) (int, error) {
	envVar := os.Getenv(varName)
	return strconv.Atoi(envVar)
}

func GetURLOrBail(urlEnvVar string) *url.URL {
	downstreamUrl := os.Getenv(urlEnvVar)
	if downstreamUrl == "" {
		log.Fatal(fmt.Errorf("No downstream URL configured. Please set %s", urlEnvVar))
	}
	parsedUrl, err := url.Parse(downstreamUrl)
	if err != nil {
		log.Fatalf("Invalid downstream url: %s. %s\n", downstreamUrl, err.Error())
	}
	return parsedUrl
}

// SetMaxIdleConnsPerHost sets the MaxIdleConnsPerHost value for the default
// HTTP transport. If you are using a custom transport, calling this function
// won't change anything.
func SetMaxIdleConnsPerHost(maxConns int) {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = maxConns
}
