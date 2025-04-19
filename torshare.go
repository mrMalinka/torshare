package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/cretz/bine/tor"
	"github.com/pkg/errors"
)

const tempFilename = "vid-compressed.mp4"

func main() {
	source := os.Args[1]
	timeoutString := os.Args[2]
	compressionLevelString := os.Args[3]
	if source == "" || timeoutString == "" || compressionLevelString == "" {
		fmt.Println("Usage:\ntorshare [filename] [timeout] [compression (0-10)]")
		os.Exit(1)
	}

	// check if user has tor installed
	// ffmpeg check is in compressMP4
	if _, err := exec.LookPath("tor"); err != nil {
		fmt.Println("`tor` not found in PATH: %w", err)
		os.Exit(1)
	}

	// check if source exists
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("File '%v' does not exist.\n", source)
		os.Exit(1)
	}
	if filepath.Ext(source) != "mp4" {
		fmt.Println("File is not an mp4.")
		os.Exit(1)
	}

	// parse timeout
	timeout, err := time.ParseDuration(timeoutString)
	if err != nil {
		fmt.Println("Failed to parse timeout duration:", err)
		os.Exit(1)
	}
	// parse compression
	compressionLevel, err := strconv.Atoi(compressionLevelString)
	if err != nil {
		fmt.Println("Failed to parse compression level:", err)
		os.Exit(1)
	}

	// generate a temporary directory and remember to close it when were done
	rootDir, closeRootDir := generateTempDir()
	defer closeRootDir()

	tempVidPath := filepath.Join(rootDir, tempFilename)
	// compress and clone the video to rootDir
	if compressionLevel != 0 {
		fmt.Println("Compressing video... (This may take a long time and consume resources)")
	}
	err = compressMP4(
		source,
		tempVidPath,
		compressionLevel,
	)
	if err != nil {
		fmt.Println("Error compressing video:", err)
		os.Exit(1)
	}
	if compressionLevel != 0 {
		uncompressedVidInfo, _ := os.Stat(source)
		compressedVidInfo, _ := os.Stat(tempVidPath)
		fmt.Printf("Uncompressed size: %v\n", prettyByteSize(uncompressedVidInfo.Size()))
		fmt.Printf("Compressed size: %v\n", prettyByteSize(compressedVidInfo.Size()))
	}

	// start tor from source directory
	fmt.Println("Connecting to tor...")
	t, err := tor.Start(context.Background(), &tor.StartConf{DataDir: rootDir})
	if err != nil {
		fmt.Println("Error starting tor:", err)
		os.Exit(1)
	}
	defer t.Close() // remember to stop tor

	// create a 30-second timeout context for the onion service
	ctxOnionTimeout, cancelOnionTimeout := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelOnionTimeout()

	// start onion service
	fmt.Println("Starting onion service...")
	onion, err := t.Listen(ctxOnionTimeout, &tor.ListenConf{
		RemotePorts: []int{80},
		Version3:    true,
	})
	if err != nil {
		fmt.Println("Error creating onion service:", err)
		os.Exit(1)
	}
	defer onion.Close() // remember to stop onion

	// make the video player at "/"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// video source is "/video"
		fmt.Fprintf(w, `
            <link href="https://vjs.zencdn.net/7.20.3/video-js.css" rel="stylesheet">
            <script src="https://vjs.zencdn.net/7.20.3/video.min.js"></script>
            <style>
                body { margin: 0; }
                video { width: 100vw; height: 100vh; }
                .video-js .vjs-big-play-button {
                    left: 50%% !important;
                    top: 50%% !important;
                    transform: translate(-50%%, -50%%) !important;
                }
            </style>
			<video
                controls
                style="margin: 0; width: 100vw; height: 100vh;"
                 id="my-video"
                class="video-js vjs-default-skin"
                controls
                width="100vw"
                height="100vh"
                data-setup='{}'
            > <source src="/video" type="video/mp4">
			</video>
		`)
	})

	// put the video source at "/video" so the main page can take from there
	http.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tempVidPath)
	})

	// start the http server
	go http.Serve(onion, nil)

	url := fmt.Sprintf("%v.onion", onion.ID)
	fmt.Printf("URL: ' %v ' \n", url)

	// copy url to clipboard
	err = clipboard.WriteAll(url)
	if err != nil {
		fmt.Println("Error copying to clipboard:", err)
	} else {
		fmt.Println("Copied to clipboard!")
	}

	// wait until timeout passes
	wait(timeout)
}
