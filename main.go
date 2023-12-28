package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const (
	youtubeApiServiceName   = "youtube"
	youtubeApiVersion       = "v3"
	defaultCommentsFileName = "comments.txt"
	invalidInputMsg         = "Invalid input. Please enter a positive integer."
	invalidURLMsg           = "Invalid YouTube URL"
	errorAPICallMsg         = "Error making search API call: %v"
	errorWritingFileMsg     = "Error writing to file: %v"
)

func getVideoId(videoUrl string) (string, error) {
	u, err := url.Parse(videoUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	videoId := q.Get("v")
	return videoId, nil
}

func getComments(videoUrls []string, maxComments int64, developerKey string) {
	var wg sync.WaitGroup
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1) // Adjust as needed

	for _, videoUrl := range videoUrls {
		wg.Add(1)
		go func(videoUrl string) {
			defer wg.Done()

			// Respect rate limit
			if err := limiter.Wait(context.Background()); err != nil { // Use the context package here
				color.Red("Rate limit error: %v", err)
				return
			}

			// Retry on network errors
			operation := func() error {
				client := &http.Client{
					Transport: &transport.APIKey{Key: developerKey},
				}

				service, err := youtube.New(client)
				if err != nil {
					return fmt.Errorf("Error creating new YouTube client: %w", err)
				}

				videoId, err := getVideoId(videoUrl)
				if err != nil {
					return fmt.Errorf("%s: %w", invalidURLMsg, err)
				}

				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("Error getting current directory: %w", err)
				}

				filename := defaultCommentsFileName + ".txt"
				commentsFile := fmt.Sprintf("%s/%s", currentDir, filename)

				file, err := os.OpenFile(commentsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("%s: %w", errorWritingFileMsg, err)
				}
				defer file.Close()

				call := service.CommentThreads.List([]string{"snippet"}).VideoId(videoId).MaxResults(maxComments)
				response, err := call.Do()
				if err != nil {
					return fmt.Errorf("%s: %w", errorAPICallMsg, err)
				}

				for _, item := range response.Items {
					comment := item.Snippet.TopLevelComment
					author := comment.Snippet.AuthorDisplayName
					text := comment.Snippet.TextDisplay
					_, err := file.WriteString(fmt.Sprintf("Comment by %s: %s\n", author, text))
					if err != nil {
						return fmt.Errorf("%s: %w", errorWritingFileMsg, err)
					}
				}

				return nil
			}

			// Use exponential backoff for retries
			err := backoff.Retry(operation, backoff.NewExponentialBackOff())
			if err != nil {
				color.Red("Failed to get comments: %v", err)
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
			color.Cyan("Configuration file not found. Creating one.")
			color.Cyan("Enter your developer key: ")
			var developerKey string
			fmt.Scanln(&developerKey)

			viper.Set("developerKey", developerKey)
			viper.WriteConfigAs("./config.yaml")

			return developerKey
		} else {
			color.Red("Error reading config file: %v", err)
			return ""
		}
	}

	return viper.GetString("developerKey")
}

func getNumberOfComments() int {
	var input string
	var maxComments int
	var err error

	for {
		color.Cyan("Enter the number of comments to retrieve: ")
		fmt.Scanln(&input)
		maxComments, err = strconv.Atoi(input)
		if err != nil || maxComments < 0 {
			color.Red(invalidInputMsg)
		} else {
			break
		}
	}

	return maxComments
}

func askToContinue() bool {
	var input string

	for {
		color.Cyan("Do you want to continue? (Y/N): ")
		fmt.Scanln(&input)
		input = strings.ToLower(input)
		if input == "y" || input == "yes" {
			return true
		} else if input == "n" || input == "no" {
			return false
		} else {
			color.Red("Invalid input. Please enter Y or N.")
		}
	}
}

func main() {
	developerKey := getDeveloperKey()

	for {
		maxComments := getNumberOfComments()

		fmt.Println("Enter the YouTube video URL: ")
		var videoUrl string
		fmt.Scanln(&videoUrl)
		videoUrls := []string{videoUrl}

		getComments(videoUrls, int64(maxComments), developerKey)

		if !askToContinue() {
			return
		}
	}
}
