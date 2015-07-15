package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/julienschmidt/httprouter"
)

type Creative struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	FileSize int64  `json:"filesize"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Mime     string `json:"mime"`
}

func upload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var response struct {
		Files []Creative `json:"files"`
		Error string     `json:"error"`
	}

	if files, err := parseUpload(w, r); err != nil {
		response.Error = fmt.Sprint(err)
	} else {
		response.Files = files
	}

	iframe := ps.ByName("iframe")

	if iframe == "" {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(response); err != nil {
			w.Write([]byte(`{"error":"could not encode json"}`))
		}
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if b, err := json.Marshal(response); err != nil {
			w.Write([]byte(`{"error":"could not encode json"}`))
		} else {
			w.Write([]byte(`'<!DOCTYPE html><html lang=en><head><meta charset=utf-8><script type="text/javascript">window.response = "`))
			w.Write(b)
			w.Write([]byte(`"';</script></head><body></body></html>`))
		}
	}
}

func parseUpload(w http.ResponseWriter, r *http.Request) ([]Creative, error) {
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	base64Buffer := bytes.Buffer{}
	base64Writer := base64.NewEncoder(base64.StdEncoding, &base64Buffer)

	size, _ := io.Copy(base64Writer, file)
	if err := base64Writer.Close(); err != nil {
		return nil, fmt.Errorf("Error reading file: %v", err)
	}

	files := make([]Creative, 0)

	if strings.ToLower(header.Filename[len(header.Filename)-4:]) == ".zip" {
		if _, err := file.Seek(0, 0); err != nil {
			return nil, err
		}

		z, err := zip.NewReader(file, size)
		if err != nil {
			return nil, fmt.Errorf("Could not read ZIP file: %v", err)
		}

		for _, f := range z.File {
			zipped, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("Could not open file in ZIP: %v", err)
			}
			defer zipped.Close()

			w, h, mime := getImageInfo(zipped)

			base64Buffer.Reset()

			size, err := io.Copy(base64Writer, zipped)
			if err != nil {
				return nil, fmt.Errorf("Could not read ZIP: %v", err)
			}

			if err := base64Writer.Close(); err != nil {
				return nil, fmt.Errorf("Could not finish ZIP: %v", err)
			}

			files = append(files, Creative{f.Name, base64Buffer.String(), size, w, h, mime})
		}
	} else {
		if _, err := file.Seek(0, 0); err != nil { // Seek to begin
			return nil, err
		}

		w, h, mime := getImageInfo(file)
		files = append(files, Creative{header.Filename, base64Buffer.String(), size, w, h, mime})
	}

	return files, nil
}

func getImageInfo(file io.Reader) (int, int, string) {
	img, mime, err := image.DecodeConfig(file)

	if err != nil {
		return 0, 0, "application/unknown"
	}

	return img.Width, img.Height, "image/" + mime
}

type notFound struct {
	http.Handler
}

func (n notFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not a valid request: %s %s", r.Method, r.URL.Path)
}

func main() {
	bind := flag.String("bind", "0.0.0.0:5241", "Bind to this IP:port combination")
	flag.Parse()

	// Set routes
	router := httprouter.New()
	router.HandleMethodNotAllowed = false
	router.POST("/upload", upload)
	router.NotFound = notFound{}

	s := &http.Server{
		Addr:           *bind,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1024 * 6, // 4k cookie + 2k http request
	}

	s.SetKeepAlivesEnabled(true)

	if err := gracehttp.Serve(s); err != nil {
		panic(err)
	}
}
