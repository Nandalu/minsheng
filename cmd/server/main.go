package main

import (
	"encoding/json"
	"fmt"
	"github.com/Nandalu/minsheng/util/jinma"
	"github.com/davecheney/gpio"
	"github.com/pkg/errors"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
)

type Server struct {
	App      string
	User     string
	GreenPin gpio.Pin
	RedPin   gpio.Pin
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

func httpHandleFunc(s *Server, path string, fn func(*Server, http.ResponseWriter, *http.Request)) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		fn(s, w, r)
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

type Reading struct {
	AQI int
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

	reading := Reading{}
	if err := json.Unmarshal([]byte(resp.Msgs[0].Body), &reading); err != nil {
		return &appError{Message: fmt.Sprintf("%+v", err)}
	}
	log.Printf("rect %+v, reading %+v", rect, reading)
	if reading.AQI > 70 {
		s.GreenPin.Clear()
		s.RedPin.Set()
	} else {
		s.RedPin.Clear()
		s.GreenPin.Set()
	}

	json.NewEncoder(w).Encode(resp)
	return nil
}

var pollutionUITmpl = template.Must(template.ParseFiles("tmpl/PollutionUI.html"))

func PollutionUI(s *Server, w http.ResponseWriter, r *http.Request) {
	rect, err := getFormRect(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("%+v", err), http.StatusBadRequest)
		return
	}
	rect.SLat = 0.06
	rect.SLng = 0.35

	resp, err := jinma.MsgsByGeoAppUser(s.App, s.User, *rect, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("%+v", err), http.StatusBadRequest)
		return
	}

	sort.Slice(resp.Msgs, func(i, j int) bool {
		iDist := math.Pow(resp.Msgs[i].Lat-rect.CLat, 2) + math.Pow(resp.Msgs[i].Lng-rect.CLng, 2)
		jDist := math.Pow(resp.Msgs[j].Lat-rect.CLat, 2) + math.Pow(resp.Msgs[j].Lng-rect.CLng, 2)
		return iDist < jDist
	})

	page := struct {
		Msgs []jinma.Msg
	}{}
	page.Msgs = resp.Msgs
	if err := pollutionUITmpl.Execute(w, page); err != nil {
		log.Printf("%+v", err)
	}
}

func Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello 民生物联网"))
}

func main() {
	server := &Server{
		App:  "16Qao77TJqiey",
		User: "12NPDF4sASbe4",
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	jsonError(server, "/Pollution", Pollution)
	httpHandleFunc(server, "/PollutionUI", PollutionUI)
	http.HandleFunc("/", Root)

	greenPin, err := gpio.OpenPin(10, gpio.ModeOutput)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	server.GreenPin = greenPin
	redPin, err := gpio.OpenPin(11, gpio.ModeOutput)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	server.RedPin = redPin
	// turn the led off on exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			fmt.Printf("\nClearing and unexporting the pin.\n")
			greenPin.Clear()
			greenPin.Close()
			redPin.Clear()
			redPin.Close()
			os.Exit(0)
		}
	}()

	port := 8080
	log.Printf("listening at %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("%+v", err)
	}
}
