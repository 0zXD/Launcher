package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ConvertToWav(inputPath string) (string, func(), error) {
	ext := strings.ToLower(filepath.Ext(inputPath))

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", nil, fmt.Errorf("ffmpeg is required to convert %s files. Please install: sudo apt install ffmpeg", ext)
	}

	tempWav := "/tmp/temp_audio_" + fmt.Sprintf("%d", os.Getpid()) + ".wav"

	fmt.Printf("Converting %s to WAV format...\n", ext)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-y",
		tempWav)

	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to convert audio file: %v", err)
	}

	fmt.Println("Conversion completed!")

	cleanup := func() {
		os.Remove(tempWav)
	}

	return tempWav, cleanup, nil
}

func GetSupportedFormats() []string {
	return []string{".wav", ".mp3", ".mp4", ".m4a", ".flac", ".ogg", ".opus", ".aac", ".wma"}
}

func IsSupported(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supported := GetSupportedFormats()

	for _, format := range supported {
		if ext == format {
			return true
		}
	}
	return false
}
