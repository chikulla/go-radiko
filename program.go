package radiko

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"time"

	"github.com/chikulla/go-radiko/internal/util"
)

// Stations is a slice of Station.
type Stations []Station

// Station is a struct.
type Station struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name"`
	Scd   Scd    `xml:"scd,omitempty"`
	Progs Progs  `xml:"progs,omitempty"`
}

type RadioStations []RadioStation

type RadioStation struct {
	ID   string `xml:"id"`
	Name string `xml:"name"`
}

// Scd is a struct.
type Scd struct {
	Progs Progs `xml:"progs"`
}

// Progs is a slice of Prog.
type Progs struct {
	Date  string `xml:"date"`
	Progs []Prog `xml:"prog"`
}

// Prog is a struct.
type Prog struct {
	Ft       string `xml:"ft,attr"`
	To       string `xml:"to,attr"`
	Ftl      string `xml:"ftl,attr"`
	Tol      string `xml:"tol,attr"`
	Dur      string `xml:"dur,attr"`
	Title    string `xml:"title"`
	SubTitle string `xml:"sub_title"`
	Desc     string `xml:"desc"`
	Pfm      string `xml:"pfm"`
	Info     string `xml:"info"`
	URL      string `xml:"url"`
}

func (c *Client) GetRadioStations(ctx context.Context) (RadioStations, error) {
	apiEndpoint := path.Join(apiV3, "station/list", fmt.Sprintf("%s.xml", c.AreaID()))

	req, err := c.newRequest(ctx, "GET", apiEndpoint, &Params{})
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d radioStationsData
	if err = decodeRadioStationsData(resp.Body, &d); err != nil {
		return nil, err
	}
	return d.radioStations(), nil
}

func (c *Client) GetProgramsByStation(ctx context.Context, stationId string, date time.Time) ([]Prog, error) {
	apiEndpoint := path.Join(apiV3, "program/station/date", util.ProgramsDate(date), fmt.Sprintf("%s.xml", stationId))
	req, err := c.newRequest(ctx, "GET", apiEndpoint, &Params{})
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d stationsData
	if err = decodeStationsData(resp.Body, &d); err != nil {
		return nil, err
	}
	return d.programs(), nil
}

func (c *Client) FindProgramByStation(ctx context.Context, stationId string, date time.Time) (*Prog, error) {
	progs, err := c.GetProgramsByStation(ctx, stationId, date)
	if err != nil {
		return nil, err
	}

	target, err := strconv.Atoi(util.Datetime(date))
	if err != nil {
		return nil, err
	}

	for _, prog := range progs {
		from, err := strconv.Atoi(prog.Ft)
		if err != nil {
			return nil, err
		}
		to, err := strconv.Atoi(prog.To)
		if err != nil {
			return nil, err
		}
		if from <= target && to > target {
			return &prog, nil
		}
	}
	return nil, errors.New("CAN'T FIND THE PROGRAM")
}

// GetStations returns the program's meta-info.
func (c *Client) GetStations(ctx context.Context, date time.Time) (Stations, error) {
	apiEndpoint := path.Join(apiV3,
		"program/date", util.ProgramsDate(date),
		fmt.Sprintf("%s.xml", c.AreaID()))

	req, err := c.newRequest(ctx, "GET", apiEndpoint, &Params{})
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d stationsData
	if err = decodeStationsData(resp.Body, &d); err != nil {
		return nil, err
	}
	return d.stations(), nil
}

// GetNowPrograms returns the program's meta-info which are currently on the air.
func (c *Client) GetNowPrograms(ctx context.Context) (Stations, error) {
	apiEndpoint := apiPath(apiV2, "program/now")

	req, err := c.newRequest(ctx, "GET", apiEndpoint, &Params{
		query: map[string]string{
			"area_id": c.AreaID(),
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d stationsData
	if err = decodeStationsData(resp.Body, &d); err != nil {
		return nil, err
	}
	return d.stations(), nil
}

// GetProgramByStartTime returns a specified program.
// This API wraps GetStations.
func (c *Client) GetProgramByStartTime(ctx context.Context, stationID string, start time.Time) (*Prog, error) {
	if stationID == "" {
		return nil, errors.New("StationID is empty")
	}

	stations, err := c.GetStations(ctx, start)
	if err != nil {
		return nil, err
	}

	ft := util.Datetime(start)
	var prog *Prog
	for _, s := range stations {
		if s.ID == stationID {
			for _, p := range s.Progs.Progs {
				if p.Ft == ft {
					prog = &p
					break
				}
			}
		}
	}
	if prog == nil {
		return nil, ErrProgramNotFound
	}
	return prog, nil
}

// GetWeeklyPrograms returns the weekly programs.
func (c *Client) GetWeeklyPrograms(ctx context.Context, stationID string) (Stations, error) {
	apiEndpoint := path.Join(apiV3,
		"program/station/weekly",
		fmt.Sprintf("%s.xml", stationID))

	req, err := c.newRequest(ctx, "GET", apiEndpoint, &Params{})
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var d stationsData
	if err = decodeStationsData(resp.Body, &d); err != nil {
		return nil, err
	}
	return d.stations(), nil
}

type radioStationsData struct {
	XMLName       xml.Name      `xml:"stations"`
	RadioStations RadioStations `xml:"station"`
}

func (d *radioStationsData) radioStations() RadioStations {
	return d.RadioStations
}

func decodeRadioStationsData(input io.Reader, stations *radioStationsData) error {
	b, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}
	if err = xml.Unmarshal(b, stations); err != nil {
		return err
	}
	return nil
}

// stationsData includes a response struct for client's users.
type stationsData struct {
	XMLName     xml.Name `xml:"radiko"`
	XMLStations struct {
		XMLName  xml.Name `xml:"stations"`
		Stations Stations `xml:"station"`
	} `xml:"stations"`
}

// stations returns Stations which is a response struct for client's users.
func (d *stationsData) stations() Stations {
	return d.XMLStations.Stations
}

func (d *stationsData) programs() []Prog {
	return d.XMLStations.Stations[0].Progs.Progs
}

// decodeStationsData parses the XML-encoded data and stores the result.
func decodeStationsData(input io.Reader, stations *stationsData) error {
	b, err := ioutil.ReadAll(input)
	if err != nil {
		return err
	}

	if err = xml.Unmarshal(b, stations); err != nil {
		return err
	}
	return nil
}
