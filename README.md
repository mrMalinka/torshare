# Torshare
Simple way to temporarily share large videos without using a 3rd party service or revealing your ip.

## Installation
Dependencies:
- [tor](https://gitlab.torproject.org/tpo/core/tor) executable **(not the tor browser)**  
- [ffmpeg](https://ffmpeg.org/) (not needed if compression isn't needed)

If you are on linux, download these using your package manager. Windows is not actively supported, but it'll probably work.
Build:
```sh
git clone https://github.com/mrMalinka/torshare
cd torshare
go build torshare.go helper.go
```

## Usage
```
torshare [filename] [timeout] [compression (0-10)]
```
eg.
```sh
torshare ./video.mp4 1h5m10s 4
```
