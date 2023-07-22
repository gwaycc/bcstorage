package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gwaylib/errors"
)

func init() {
	RegisterHandle("/file/upload", uploadHandler)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	authPath, _, err := authWrite(r)
	if err != nil {
		return writeMsg(w, 401, errors.As(err).Code())
	}

	to := authPath
	posStr := r.FormValue("pos")
	pos, _ := strconv.ParseInt(posStr, 10, 64)
	dir := filepath.Dir(to)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return writeMsg(w, 500, err.Error())
	}
	toFile, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return writeMsg(w, 403, errors.As(err, to).Error())
	}
	defer toFile.Close()

	toStat, err := toFile.Stat()
	if err != nil {
		return writeMsg(w, 500, errors.As(err).Error())
	}
	if time.Now().Sub(toStat.ModTime()) > 24*time.Hour {
		return writeMsg(w, 403, "file has been locked")
	}

	if _, err := toFile.Seek(pos, 0); err != nil {
		return writeMsg(w, 500, errors.As(err).Error())
	}
	if _, err := io.Copy(toFile, r.Body); err != nil {
		return writeMsg(w, 500, errors.As(err).Error())
	}
	// flush the data?
	toFile.Close()
	r.Body.Close()

	log.Infof("Upload file %s, offset:%d, from %s", to, pos, r.RemoteAddr)
	if r.FormValue("checksum") == "sha1" {
		toFile, err = os.Open(to)
		if err != nil {
			return writeMsg(w, 403, errors.As(err, to).Error())
		}
		if _, err := toFile.Seek(0, 0); err != nil {
			return writeMsg(w, 500, errors.As(err).Error())
		}
		ah := sha1.New()
		if _, err := io.Copy(ah, toFile); err != nil {
			return writeMsg(w, 500, errors.As(err).Error())
		}
		aSum := ah.Sum(nil)
		return writeMsg(w, 200, fmt.Sprintf("%x", aSum))
	}
	return writeMsg(w, 200, "success")
}
