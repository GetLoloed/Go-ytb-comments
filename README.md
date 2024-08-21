# YouTube Comment Retriever

This Go application allows users to retrieve comments from YouTube videos using the YouTube Data API v3.

## Features

- Retrieve a specified number of comments from a YouTube video
- Save comments to a text file named after the video ID
- Rate limiting to comply with API usage restrictions
- Exponential backoff for error handling
- Configuration management using Viper
- Colorized console output for better user experience

## Prerequisites

- Go 1.15 or higher
- A Google Developer API key with YouTube Data API v3 enabled

## Installation

1. Clone this repository
2. Install dependencies:
   ```
   go mod tidy
   ```

## Usage

1. Run the application:
   ```
   go run main.go
   ```

2. If it's your first time running the app, you'll be prompted to enter your YouTube Data API key. This will be saved in a `config.yaml` file for future use.

3. Enter the number of comments you want to retrieve when prompted.

4. Paste the YouTube video URL when asked.

5. The application will retrieve the comments and save them to a file in the current directory.

6. You'll be asked if you want to continue retrieving comments for other videos.

## Configuration

The application uses a `config.yaml` file to store the YouTube Data API key. If the file doesn't exist, you'll be prompted to enter the key, and it will be created automatically.

## Error Handling

The application includes error handling for various scenarios, including:
- Invalid input
- API rate limiting
- Network issues
- File writing errors

## License

[MIT License](LICENSE)

## Disclaimer

This application is for educational purposes only. Make sure to comply with YouTube's terms of service and API usage guidelines when using this tool.
