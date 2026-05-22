package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	repoOwner = "FallenProjects"
	repoName  = "tdlib-build"
)

var versionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

func resolveVersion(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("please provide a version")
	}

	userVersion := strings.TrimSpace(args[0])
	if userVersion == "" {
		return "", errors.New("please provide a version")
	}
	if !versionPattern.MatchString(userVersion) {
		return "", fmt.Errorf("invalid version format: %q (expected version like vX.Y.Z)", userVersion)
	}

	if !strings.HasPrefix(userVersion, "v") {
		userVersion = "v" + userVersion
	}

	return userVersion, nil
}

// writeFile writes from an io.Reader to a file named name and ensures the file
// is closed on all paths.
func writeFile(name string, r io.Reader) error {
	outFile, err := os.Create(name)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, r)
	return err
}

// writeFileFromReadCloser writes from an io.ReadCloser to a file and ensures both
// the reader and file are closed on all paths.
func writeFileFromReadCloser(name string, rc io.ReadCloser) error {
	defer rc.Close()
	outFile, err := os.Create(name)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func main() {
	targetVersion, err := resolveVersion(os.Args[1:])
	if err != nil {
		panic(err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var filename string
	var matchFunc func(string) bool

	switch goos {
	case "linux":
		arch := goarch
		if arch == "amd64" {
			arch = "x86_64"
		} else if arch == "arm64" {
			arch = "aarch64"
		}
		filename = fmt.Sprintf("TDLib-tdjson-linux-%s.tar.gz", arch)
		matchFunc = func(name string) bool {
			return strings.HasPrefix(name, "libtdjson.so")
		}
	case "darwin":
		arch := goarch
		if arch == "amd64" {
			arch = "x86_64"
		}
		filename = fmt.Sprintf("TDLib-tdjson-macos-%s.tar.gz", arch)
		matchFunc = func(name string) bool {
			return strings.HasPrefix(name, "libtdjson") && strings.HasSuffix(name, ".dylib")
		}
	case "windows":
		arch := goarch
		if arch == "amd64" {
			arch = "x64"
		} else if arch == "386" {
			arch = "x86"
		}
		filename = fmt.Sprintf("TDLib-tdjson-windows-%s.zip", arch)
		matchFunc = func(name string) bool {
			return strings.HasSuffix(name, ".dll")
		}
	default:
		panic("unsupported OS: " + goos + "please build TDLib manually following https://tdlib.github.io/td/build.html?language=Go")
	}

	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", repoOwner, repoName, targetVersion, filename)

	fmt.Printf("Downloading %s (%s) from %s...\n", filename, targetVersion, downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			panic(fmt.Sprintf("release not found for version %s (%s)", targetVersion, resp.Status))
		}
		panic(fmt.Sprintf("failed to download %s: %s", filename, resp.Status))
	}

	tmpFile, err := os.CreateTemp("", "tdjson-archive-*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		panic(err)
	}

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("Extracting...")
	found := false

	if strings.HasSuffix(filename, ".tar.gz") {
		gz, err := gzip.NewReader(tmpFile)
		if err != nil {
			panic(err)
		}
		defer gz.Close()

		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}

			name := filepath.Base(hdr.Name)
			if matchFunc(name) {
				if err := writeFile(name, tr); err != nil {
					panic(err)
				}
				fmt.Println("Extracted:", name)
				found = true
			}
		}
	} else if strings.HasSuffix(filename, ".zip") {
		stat, err := tmpFile.Stat()
		if err != nil {
			panic(err)
		}
		zr, err := zip.NewReader(tmpFile, stat.Size())
		if err != nil {
			panic(err)
		}

		for _, f := range zr.File {
			name := filepath.Base(f.Name)
			if matchFunc(name) {
				rc, err := f.Open()
				if err != nil {
					panic(err)
				}

				if err := writeFileFromReadCloser(name, rc); err != nil {
					panic(err)
				}

				fmt.Println("Extracted:", name)
				found = true
			}
		}
	}

	if !found {
		panic("no matching library files found in archive")
	}
	fmt.Println("Done.")
}
