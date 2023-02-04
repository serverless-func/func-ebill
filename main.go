package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type fetchConfig struct {
	Username string `form:"username"`
	Password string `form:"password"`
	Hour     int64  `form:"hour"`
}

// func main() {
// 	http.HandleFunc("/", Handler)
// 	log.Fatal(http.ListenAndServe(":9000", nil))
// }

// Handler is the entry point for fission function
func Handler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("Action")
	switch action {
	case "":
		writeData(w, http.StatusOK, "text/plain; charset=utf-8", []byte("it works"))
	case "Ping":
		writeData(w, http.StatusOK, "text/plain; charset=utf-8", []byte("pong"))
	case "Cmb":
		var cfg fetchConfig
		err := json.NewDecoder(r.Body).Decode(&cfg)
		if err != nil {
			writeJsonFail(w, "missing required body")
			return
		}
		if cfg.Hour == 0 {
			cfg.Hour = 24
		}
		orders, err := emailParseCmb(cfg)
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		writeJsonData(w, orders)
	case "FileCmb":
		file, fh, err := r.FormFile("file")
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		localfilepath := "/tmp/" + filepath.Base(fh.Filename)
		localfile, err := os.OpenFile(localfilepath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		_, err = io.Copy(localfile, file)
		localfile.Close()
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		defer func() {
			_ = os.Remove(localfilepath)
		}()
		orders, err := fileParseCmb(localfilepath)
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}

		writeJsonData(w, orders)
	case "FileSpdb":
		file, fh, err := r.FormFile("file")
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		localfilepath := "/tmp/" + filepath.Base(fh.Filename)
		localfile, err := os.OpenFile(localfilepath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		_, err = io.Copy(localfile, file)
		localfile.Close()
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}
		defer func() {
			_ = os.Remove(localfilepath)
		}()

		orders, err := fileParseSpdb(localfilepath, r.FormValue("password"))
		if err != nil {
			writeJsonFail(w, err.Error())
			return
		}

		writeJsonData(w, orders)

	case "Debug":
		writeJsonData(w, r.Header)
	}
}

func writeData(w http.ResponseWriter, code int, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(code)
	w.Write(data)
}

func writeJsonData(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body := make(map[string]interface{})
	body["msg"] = "success"
	body["data"] = data
	body["timestamp"] = time.Now().Unix()
	json.NewEncoder(w).Encode(body)
}

func writeJsonFail(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	body := make(map[string]interface{})
	body["msg"] = msg
	body["timestamp"] = time.Now().Unix()
	json.NewEncoder(w).Encode(body)
}
