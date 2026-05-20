// This program downloads the dist assets for the current swagger-ui version and places them into the embed directory

//go:build ignore
// +build ignore

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const assetsDir = "assets"

type releaseResp []struct {
	// TagName is a release tag name
	TagName string `json:"tag_name"`
}

const requestTimeout = 30 * time.Second

func main() {
	log.SetFlags(0)
	releases := releaseResp{}
	client := &http.Client{Timeout: requestTimeout}

	// get the releases so we can download the latest one
	req, err := http.NewRequest("GET", "https://api.github.com/repos/swagger-api/swagger-ui/releases", nil)
	if err != nil {
		log.Fatalf("error creating release list request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error getting release list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("got status [%s] on release list download", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		log.Fatalf("error decoding response: %v", err)
	}
	if len(releases) == 0 {
		log.Fatal("somehow got no releases, nothing to do")
	}
	tag := releases[0].TagName

	current, err := ioutil.ReadFile("current_version.txt")
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// no problem, just do it
		default:
			log.Fatalf("unable to check version in current_version.txt: %v", err)
		}
	}
	cv := string(bytes.TrimRight(current, "\n"))

	if cv == tag {
		log.Print("version is current, nothing to do")
		os.Exit(0)
	}

	log.Printf("downloading release %s...", tag)

	resp, err = client.Get(fmt.Sprintf("https://github.com/swagger-api/swagger-ui/archive/%s.tar.gz", tag))
	if err != nil {
		log.Fatalf("error downloading release archive: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("got status [%s] on release archive download", resp.Status)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatalf("error opening file as gzip: %v", err)
	}
	if err := os.RemoveAll(assetsDir); err != nil {
		log.Fatalf("error removing old embed directory")
	}
	if err := os.Mkdir(assetsDir, 0o700); err != nil {
		log.Fatalf("error recreating embed directory")
	}
	tr := tar.NewReader(zr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar parsing error: %v", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		outPath, ok, err := distOutputPath(header.Name)
		if err != nil {
			log.Fatalf("unsafe archive path %q: %v", header.Name, err)
		}
		if !ok {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			log.Fatalf("error creating output directory for %s: %v", outPath, err)
		}
		out, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("error create output file: %v", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			log.Fatalf("error writing output file: %v", err)
		}
		if err := out.Close(); err != nil {
			log.Fatalf("error closing output file %s: %v", outPath, err)
		}
	}
	// replace the hard-coded JSON file with a generic file and disable the topbar
	initFile, err := os.ReadFile(filepath.Join(assetsDir, "swagger-initializer.js"))
	if err != nil {
		log.Fatalf("error opening swagger-initializer.js for templating :%v", err)
	}
	// replace the url
	//newInit := strings.Replace(
	//string(initFile),
	//`url: "https://petstore.swagger.io/v2/swagger.json"`,
	//`url: "./swagger_spec"`,
	//-1,
	//)
	newInit := regexp.MustCompile(`url:\s+"[^"]*"`).ReplaceAllLiteral(initFile, []byte(`url: "./swagger_spec"`))
	newInit = regexp.MustCompile(`,?\s+SwaggerUIStandalonePreset.*\n`).ReplaceAllLiteral(newInit, []byte("\n"))
	newInit = regexp.MustCompile(`(?s),\s+plugins: \[.*],\n`).ReplaceAllLiteral(newInit, []byte("\n"))
	newInit = regexp.MustCompile(`\n\s*layout:.*\n`).ReplaceAllLiteral(newInit, []byte("\n"))
	//fmt.Println(string(newInit))
	newinitFile, err := os.Create(filepath.Join(assetsDir, "swagger-initializer.js"))
	if err != nil {
		log.Fatalf("error re-creating swagger-initializer.js file: %v", err)
	}
	defer newinitFile.Close()
	if _, err := newinitFile.Write(newInit); err != nil {
		log.Fatalf("unable to write to swagger-initializer.js: %v", err)
	}
	newcv, err := os.Create("current_version.txt")
	if err != nil {
		log.Fatalf("can't update current_version.txt: %v", err)
	}
	defer newcv.Close()
	newcv.WriteString(tag)
	log.Printf("updated swaggerui from %s => %s, please check templated swagger-initializer.js and retag repo", cv, tag)
}

func distOutputPath(headerName string) (string, bool, error) {
	_, rel, ok := strings.Cut(headerName, "/")
	if !ok {
		return "", false, nil
	}
	if !strings.HasPrefix(rel, "dist/") {
		return "", false, nil
	}

	rel = strings.TrimPrefix(rel, "dist/")
	clean := path.Clean(rel)
	if clean == "." {
		return "", false, nil
	}
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false, fmt.Errorf("path escapes dist directory")
	}

	return filepath.Join(assetsDir, filepath.FromSlash(clean)), true, nil
}
