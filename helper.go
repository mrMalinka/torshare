package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func wait(timeout time.Duration) bool {
	yesChan := make(chan bool, 1)
	inputReader := bufio.NewReader(os.Stdin)

	go func() {
		fmt.Printf("Enter 'stop' to stop> ")
		text, _ := inputReader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(text)) == "stop" {
			yesChan <- true
		}
	}()

	select {
	case <-yesChan:
		return true
	case <-time.After(timeout):
		fmt.Println("\nStopped")
		return false
	}
}

func compressMP4(inputPath, outputPath string, level int) error {
	if level < 0 || level > 10 {
		return fmt.Errorf("compression level must be between 0 and 10")
	}

	if level == 0 {
		// simply hard link the file if no compression is needed
		return os.Link(inputPath, outputPath)
	} else {
		crf := 18 + int(math.Round(float64(level-1)*3.6667))
		cmd := exec.Command(
			"ffmpeg",
			"-y",            // overwrite output
			"-i", inputPath, // input file
			"-vcodec", "libx264", // video codec
			"-crf", strconv.Itoa(crf), // compression level (18–51)
			"-preset", "slow", // slower preset for better compression
			"-c:a", "copy", // copy audio
			outputPath,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg execution failed: %w", err)
		}
	}

	return nil
}

func generateTempDir() (string, func()) {
	const prefix = "torshare-"
	name, err := os.MkdirTemp(os.TempDir(), prefix)
	if err != nil {
		log.Fatalln("Error creating temporary directory:", err)
	}

	return name, func() {
		err := os.RemoveAll(name)
		if err != nil {
			log.Fatalln("Error closing temporary directory:", err)
		}
	}
}

func prettyByteSize(size int64) string {
	if size == 0 {
		return "0 B"
	}

	base := 1000.0
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	exp := math.Log(float64(size)) / math.Log(base)
	exp = math.Floor(exp)

	if exp < 0 {
		exp = 0
	} else if exp >= float64(len(units)) {
		exp = float64(len(units) - 1)
	}

	value := float64(size) / math.Pow(base, exp)
	unit := units[int(exp)]

	formatted := fmt.Sprintf("%.2f", value)
	formatted = strings.TrimSuffix(formatted, ".00")
	formatted = strings.TrimSuffix(formatted, ".0")

	return fmt.Sprintf("%s %s", formatted, unit)
}
