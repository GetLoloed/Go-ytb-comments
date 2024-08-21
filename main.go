package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const (
	youtubeApiServiceName   = "youtube"
	youtubeApiVersion       = "v3"
	defaultCommentsFileName = "comments"
	invalidInputMsg         = "Invalid input. Please enter a positive integer."
	invalidURLMsg           = "Invalid YouTube URL"
	errorAPICallMsg         = "Error during API search call: %v"
	errorWritingFileMsg     = "Error writing to file: %v"
)

func getVideoId(videoUrl string) (string, error) {
	u, err := url.Parse(videoUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	return u.Query().Get("v"), nil
}

func getComments(ctx context.Context, videoUrls []string, maxComments int64, developerKey string) {
	var wg sync.WaitGroup
	limiter := rate.NewLimiter(rate.Every(time.Second), 1)

	for _, videoUrl := range videoUrls {
		wg.Add(1)
		go func(videoUrl string) {
			defer wg.Done()

			if err := limiter.Wait(ctx); err != nil {
				color.Red("Rate limit error: %v", err)
				return
			}

			operation := func() error {
				client := &http.Client{
					Transport: &transport.APIKey{Key: developerKey},
				}

				service, err := youtube.New(client)
				if err != nil {
					return fmt.Errorf("error creating new YouTube client: %w", err)
				}

				videoId, err := getVideoId(videoUrl)
				if err != nil {
					return fmt.Errorf("%s: %w", invalidURLMsg, err)
				}

				filename := fmt.Sprintf("%s_%s.txt", defaultCommentsFileName, videoId)
				file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("%s: %w", errorWritingFileMsg, err)
				}
				defer file.Close()

				writer := bufio.NewWriter(file)
				defer writer.Flush()

				call := service.CommentThreads.List([]string{"snippet"}).VideoId(videoId).MaxResults(maxComments)
				response, err := call.Do()
				if err != nil {
					return fmt.Errorf("%s: %w", errorAPICallMsg, err)
				}

				for _, item := range response.Items {
					comment := item.Snippet.TopLevelComment
					_, err := fmt.Fprintf(writer, "Comment from %s: %s\n", comment.Snippet.AuthorDisplayName, comment.Snippet.TextDisplay)
					if err != nil {
						return fmt.Errorf("%s: %w", errorWritingFileMsg, err)
					}
				}

				return nil
			}

			err := backoff.Retry(operation, backoff.NewExponentialBackOff())
			if err != nil {
				color.Red("Failed to retrieve comments: %v", err)
			}
		}(videoUrl)
	}

	wg.Wait()
}

func getDeveloperKey() string {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			color.Cyan("Configuration file not found. Creating.")
			color.Cyan("Enter your developer key: ")
			developerKey := readInput()

			viper.Set("developerKey", developerKey)
			if err := viper.WriteConfigAs("./config.yaml"); err != nil {
				color.Red("Error writing configuration file: %v", err)
			}

			return developerKey
		} else {
			color.Red("Error reading configuration file: %v", err)
			return ""
		}
	}

	return viper.GetString("developerKey")
}

func getNumberOfComments() int {
	for {
		color.Cyan("Enter the number of comments to retrieve: ")
		input := readInput()
		maxComments, err := strconv.Atoi(input)
		if err != nil || maxComments < 0 {
			color.Red(invalidInputMsg)
		} else {
			return maxComments
		}
	}
}

func askToContinue() bool {
	for {
		color.Cyan("Do you want to continue? (Y/N): ")
		input := strings.ToLower(readInput())
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			color.Red("Invalid input. Please enter Y or N.")
		}
	}
}

func readInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func main() {
	developerKey := getDeveloperKey()
	ctx := context.Background()

	for {
		maxComments := getNumberOfComments()

		color.Cyan("Enter the YouTube video URL: ")
		videoUrl := readInput()
		videoUrls := []string{videoUrl}

		getComments(ctx, videoUrls, int64(maxComments), developerKey)

		if !askToContinue() {
			return
		}
	}
}
