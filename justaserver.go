package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"speech-to-text/converter"
)

const (
	SERVER_ACCESS_TOKEN = "4HSFWG4UCYTYED2AG6C43D42SQKZ4RSX"
	CLIENT_ACCESS_TOKEN = "2JE2IXGWBZAJF3OMXIVXQQ4FKBR3NYCE"
	WIT_API_URL         = "https://api.wit.ai/speech"
)

type Token struct {
	Token      string  `json:"token"`
	Confidence float64 `json:"confidence"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
}

type Speech struct {
	Confidence float64 `json:"confidence"`
	Tokens     []Token `json:"tokens"`
}

type WitResponse struct {
	Speech Speech `json:"speech"`
	Text   string `json:"text"`
	Type   string `json:"type"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run justaserver.go <audio_file_path>")
		fmt.Println("Example: go run justaserver.go /path/to/audio.wav")
		fmt.Printf("Supported formats: %v\n", converter.GetSupportedFormats())
		return
	}

	audioFilePath := os.Args[1]

	if _, err := os.Stat(audioFilePath); os.IsNotExist(err) {
		log.Fatalf("Audio file does not exist: %s", audioFilePath)
	}

	if !converter.IsSupported(audioFilePath) {
		log.Fatalf("Unsupported file format. Supported formats: %v", converter.GetSupportedFormats())
	}

	fmt.Printf("Processing audio file: %s\n", audioFilePath)

	fmt.Println("Converting to proper WAV format...")
	wavFilePath, cleanup, err := converter.ConvertToWav(audioFilePath)
	if err != nil {
		log.Fatalf("Error converting audio: %v", err)
	}
	defer cleanup()

	response, err := sendAudioToWit(wavFilePath)
	if err != nil {
		log.Fatalf("Error calling Wit.AI API: %v", err)
	}

	parseFinalTranscription(response)
}

func sendAudioToWit(audioFilePath string) (string, error) {
	file, err := os.Open(audioFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	client := &http.Client{}

	req, err := http.NewRequest("POST", WIT_API_URL, file)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+SERVER_ACCESS_TOKEN)
	req.Header.Set("Content-Type", "audio/wav")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func parseFinalTranscription(response string) {
	var finalTranscription WitResponse
	finalText := ""

	decoder := json.NewDecoder(strings.NewReader(response))

	for decoder.More() {
		var resp WitResponse
		if err := decoder.Decode(&resp); err != nil {
			continue
		}

		if resp.Type == "FINAL_TRANSCRIPTION" {
			finalTranscription = resp
			finalText = resp.Text
			break
		}
	}

	if finalText == "" {
		decoder = json.NewDecoder(strings.NewReader(response))
		for decoder.More() {
			var resp WitResponse

			if err := decoder.Decode(&resp); err != nil {
				continue
			}

			if len(resp.Speech.Tokens) > 0 && resp.Text != "" {
				finalTranscription = resp
				fmt.Printf("Decoding response: %s\n", response)
				finalText = resp.Text
			}
		}

		if finalText == "" {
			fmt.Println("No transcription with speech data found")
			return
		}
	}

	for _, token := range finalTranscription.Speech.Tokens {
		fmt.Printf("{\n")
		fmt.Printf("  word=\"%s\"\n", token.Token)
		fmt.Printf("  confidence=%.4f\n", token.Confidence)
		fmt.Printf("}\n\n")
	}

	re := regexp.MustCompile(`\s+`)
	cleanedText := re.ReplaceAllString(finalText, " ")
	cleanedText = strings.TrimSpace(cleanedText)

	fmt.Printf("\"%s\"\n", cleanedText)
}
