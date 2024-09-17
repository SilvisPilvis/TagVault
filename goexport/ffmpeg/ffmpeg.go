package ffmpeg

import (
	"os/exec"
	"path/filepath"

	"fmt"
	"io"
	"net/http"
	"os"
)

// func DownloadFFmpeg() error {
// 	return exec.Command("ffmpeg", "-version").Run()
// }

// func getImagePath() string {
// 	userHome, _ := os.UserHomeDir()
// 	if runtime.GOOS == "linux" {
// 		return userHome + "/Attēli/wallpapers/"
// 	}
// 	return `C:\Users\Silvestrs\Desktop\test`
// }

func DownloadFFmpegLinux() error {
	resp, err := http.Get("https://johnvansickle.com/ffmpeg/builds/ffmpeg-git-amd64-static.tar.xz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// fmt.Println(resp.Header.Get(""))
	// return nil

	// Create the file
	out, err := os.Create(filepath.Base(resp.Request.URL.Path))
	if err != nil {
		return err
	}
	defer out.Close()

	// // Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// // Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFFmpegWindows() error {
	resp, err := http.Get("https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-full.7z")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath.Base(resp.Request.URL.Path))
	if err != nil {
		return err
	}
	defer out.Close()

	// // Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// // Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func Convert(input, output string) error {
	// return exec.Command("ffmpeg", "-i", input, "-vf", "scale=320:-1", output).Run()
	return exec.Command("ffmpeg", "-i", input, " ", output).Run()
}
