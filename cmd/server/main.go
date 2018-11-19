package main

import (
	"encoding/json"
	"fmt"
	"github.com/Nandalu/minsheng/util/jinma"
	"github.com/pkg/errors"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
)

type Server struct {
	App  string
	User string
}

type appError struct {
	Code    string
	Message string
}

func (a *appError) Error() string {
	return fmt.Sprintf("%s: %s", a.Code, a.Message)
}

func jsonError(s *Server, path string, fn func(*Server, http.ResponseWriter, *http.Request) *appError) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		appErr := fn(s, w, r)
		if appErr != nil {
			je := struct {
				Error struct {
					Code    string `json:",omitempty"`
					Message string
				}
			}{}
			je.Error.Code = appErr.Code
			je.Error.Message = appErr.Message
			b, _ := json.Marshal(&je)
			http.Error(w, string(b), http.StatusBadRequest)
		}
	})
}

func getFormRect(r *http.Request) (*jinma.Rect, error) {
	clat, err := strconv.ParseFloat(r.FormValue("CLat"), 64)
	if err != nil {
		return nil, errors.Wrap(err, "CLat")
	}
	clng, err := strconv.ParseFloat(r.FormValue("CLng"), 64)
	if err != nil {
		return nil, errors.Wrap(err, "CLng")
	}
	rect := &jinma.Rect{
		CLat: clat,
		CLng: clng,
	}
	return rect, nil
}

func Pollution(s *Server, w http.ResponseWriter, r *http.Request) *appError {
	rect, err := getFormRect(r)
	if err != nil {
		return &appError{Message: fmt.Sprintf("%+v", err)}
	}
	rect.SLat = 0.06
	rect.SLng = 0.35

	resp, err := jinma.MsgsByGeoAppUser(s.App, s.User, *rect, nil)
	if err != nil {
		return &appError{Message: fmt.Sprintf("%+v", err)}
	}

	sort.Slice(resp.Msgs, func(i, j int) bool {
		iDist := math.Pow(resp.Msgs[i].Lat-rect.CLat, 2) + math.Pow(resp.Msgs[i].Lng-rect.CLng, 2)
		jDist := math.Pow(resp.Msgs[j].Lat-rect.CLat, 2) + math.Pow(resp.Msgs[j].Lng-rect.CLng, 2)
		return iDist < jDist
	})

	json.NewEncoder(w).Encode(resp)
	return nil
}

func Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello 民生物联网"))
}

func main() {
	server := &Server{
		App:  "16Qao77TJqiey",
		User: "12NPDF4sASbe4",
	}
	jsonError(server, "/Pollution", Pollution)
	http.HandleFunc("/", Root)

	port := 8080
	log.Printf("listening at %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("%+v", err)
	}
}
