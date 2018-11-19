package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Nandalu/minsheng/util"
	"github.com/Nandalu/minsheng/util/jinma"
	"github.com/pkg/errors"
	"log"
	"math/rand"
	"strconv"
	"time"
)

var (
	jinmaToken string
	randomSeed int64
)

func init() {
	flag.StringVar(&jinmaToken, "jinmaToken", "", "Jinma user token")
	flag.Int64Var(&randomSeed, "randomSeed", 0, "random seed")
}

type Pollution struct {
	SiteName    string `json:",omitempty"`
	County      string `json:",omitempty"`
	AQI         int
	Pollutant   string
	Status      string
	PublishTime int64   `json:",omitempty"`
	Latitude    float64 `json:",omitempty"`
	Longitude   float64 `json:",omitempty"`
}

func (p *Pollution) UnmarshalJSON(b []byte) error {
	sm := make(map[string]string)
	if err := json.Unmarshal(b, &sm); err != nil {
		return errors.Wrap(err, "Unmarshal")
	}

	t, err := time.Parse("2006-01-02 15:04", sm["PublishTime"])
	if err != nil {
		return errors.Wrap(err, "Parse")
	}

	aqi, err := strconv.Atoi(sm["AQI"])
	if err != nil {
		return errors.Wrap(err, sm["AQI"])
	}

	lat, err := strconv.ParseFloat(sm["Latitude"], 64)
	if err != nil {
		return errors.Wrap(err, sm["Latitude"])
	}
	lng, err := strconv.ParseFloat(sm["Longitude"], 64)
	if err != nil {
		return errors.Wrap(err, sm["Longitude"])
	}

	p.SiteName = sm["SiteName"]
	p.County = sm["County"]
	p.AQI = aqi
	p.Pollutant = sm["Pollutant"]
	p.Status = sm["Status"]
	p.PublishTime = t.Unix()
	p.Latitude = lat
	p.Longitude = lng

	if p.SiteName == "" {
		return fmt.Errorf("no SiteName")
	}
	if p.County == "" {
		return fmt.Errorf("no County")
	}
	if p.Status == "" {
		return fmt.Errorf("no Status")
	}

	return nil
}

func getPollution() ([]Pollution, error) {
	urlStr := "https://opendata.epa.gov.tw/ws/Data/AQI/?$format=json"
	resp, body, err := util.JSONReq3("GET", urlStr, nil)
	if err != nil {
		return nil, errors.Wrap(err, "JSONReq3")
	}
	if resp.StatusCode != 200 {
		return nil, errors.Wrap(err, fmt.Sprintf("%+v", resp))
	}

	res := []Pollution{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.Wrap(err, string(body))
	}

	return res, nil
}

func create(inP Pollution) (*jinma.Msg, error) {
	// Use the time as the sortkey.
	// To avoid collided sortkeys, randomly a time interval.
	skf64 := float64(inP.PublishTime)
	skf64 += float64(rand.Intn(60*60 - 1))
	skf64 += rand.Float64()

	tStr := time.Unix(inP.PublishTime, 0).UTC().Format("20060102_1504")
	customID := fmt.Sprintf("%s%s%s", inP.County, inP.SiteName, tStr)
	if customID == "" {
		return nil, fmt.Errorf("empty customID for %+v", inP)
	}

	// Make a copy of the transaction and remove the unneeded fields.
	p := inP
	// These fields are unneeded because they are contained in the jinma.Msg itself.
	p.SiteName = ""
	p.County = ""
	p.PublishTime = 0
	p.Latitude = 0
	p.Longitude = 0
	pbody, err := json.Marshal(p)
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}

	msg, err := jinma.MsgCreate(jinmaToken, string(pbody), inP.Latitude, inP.Longitude, &skf64, customID)
	if err != nil {
		return nil, errors.Wrap(err, "jinma.MsgCreate")
	}
	return msg, nil
}

func main() {
	flag.Parse()
	rand.Seed(randomSeed)

	pollution, err := getPollution()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	for i, p := range pollution {
		msg, err := create(p)
		if err != nil {
			log.Fatalf("%+v", err)
		}
		log.Printf("%d %+v", i, msg)
	}
}
