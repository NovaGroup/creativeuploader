package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/julienschmidt/httprouter"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"
)

const bind = "0.0.0.0:5241"

func upload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	type Creative struct {
		Name     string `json:"name"`
		Content  string `json:"content"`
		FileSize int64  `json:"filesize"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Mime     string `json:"mime"`
	}

	var response struct {
		Files []Creative `json:"files"`
		Error string     `json:"error"`
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error = "No file parameter in POST request"
	} else {
		defer file.Close()

		base64Buffer := bytes.Buffer{}
		base64Writer := base64.NewEncoder(base64.StdEncoding, &base64Buffer)

		size, _ := io.Copy(base64Writer, file)
		if err := base64Writer.Close(); err != nil {
			response.Error = "Error reading file: " + err.Error()
		} else {
			if strings.ToLower(header.Filename[len(header.Filename)-4:]) == ".zip" {
				if _, err := file.Seek(0, 0); err != nil {
					response.Error = err.Error()
				} else {
					z, err := zip.NewReader(file, size)
					if err != nil {
						response.Error = "Could not read ZIP file: " + err.Error()
					} else {
						for _, f := range z.File {
							if zipped, err := f.Open(); err != nil {
								response.Error = "Could not open file in ZIP: " + err.Error()
							} else {
								defer zipped.Close()
								w, h, mime := getImageInfo(zipped)
								base64Buffer.Reset()
								size, _ := io.Copy(base64Writer, zipped)
								base64Writer.Close()
								response.Files = append(response.Files, Creative{f.Name, base64Buffer.String(), size, w, h, mime})
							}
						}
					}
				}
			} else {
				file.Seek(0, 0) // Seek to begin
				w, h, mime := getImageInfo(file)
				response.Files = append(response.Files, Creative{header.Filename, base64Buffer.String(), size, w, h, mime})
			}
		}
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

func getImageInfo(file io.Reader) (int, int, string) {
	img, mime, err := image.DecodeConfig(file)

	if err != nil {
		return 0, 0, "application/unknown"
	}

	return img.Width, img.Height, "image/" + mime
}

func emptypage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not a valid request: %s %s", r.Method, r.URL.Path)
}

func main() {
	fmt.Println("Starting web server")

	// Set routes
	router := httprouter.New()
	router.HandleMethodNotAllowed = false
	router.POST("/upload", upload)
	router.NotFound = emptypage

	s := &http.Server{
		Addr:           bind,
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
