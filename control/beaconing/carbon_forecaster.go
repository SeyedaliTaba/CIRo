package beaconing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/scionproto/scion/pkg/private/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/scionproto/scion/control/ifstate"
	"github.com/scionproto/scion/private/periodic"
)

var _ periodic.Task = (*CarbonForecaster)(nil)

type CarbonForecaster struct {
	ElectricityMapsAPIKeyFile string
	GeolocationFile           string
	CarbonForecastFile        string
	AllInterfaces             *ifstate.Interfaces
	Tick                      Tick
	Success                   bool
	ForecastAvailable         bool
}

type ForecastType struct {
	CarbonIntensity uint32 `json:"carbonIntensity"`
	Datetime        string `json:"datetime"`
}

type ElectricityMapForecastResponse struct {
	Zone      string         `json:"zone"`
	Forecast  []ForecastType `json:"forecast"`
	UpdatedAt string         `json:"updatedAt"`
}

type HistoryType struct {
	Zone               string `json:"zone"`
	CarbonIntensity    uint32 `json:"carbonIntensity"`
	Datetime           string `json:"datetime"`
	UpdatedAt          string `json:"updatedAt"`
	CreatedAt          string `json:"createdAt"`
	EmissionFactorType string `json:"emissionFactorType"`
	IsEstimated        bool   `json:"isEstimated"`
	EstimationMethod   string `json:"estimationMethod"`
}

type ElectricityMapHistoryResponse struct {
	Zone    string        `json:"zone"`
	History []HistoryType `json:"history"`
}

type Geolocation struct {
	Zone string `json:"zone"`
}

type ElectricityMapsAPIKey struct {
	Key string `json:"key"`
}

func (c *CarbonForecaster) Name() string {
	return "control_beaconing_carbon_forecaster"
}

func (c *CarbonForecaster) Run(ctx context.Context) {
	c.Tick.SetNow(time.Now())
	c.ForecastCarbon(ctx)
	c.Tick.UpdateLast()
}

func (c *CarbonForecaster) GetAPIKey(ctx context.Context) string {
	theLogger := withSilent(ctx, false)
	raw, err := os.ReadFile(c.ElectricityMapsAPIKeyFile)
	if err != nil {
		theLogger.Error("failed to read api key file ", c.ElectricityMapsAPIKeyFile, err)
	}
	var key ElectricityMapsAPIKey
	if err = json.Unmarshal(raw, &key); err != nil {
		theLogger.Error("failed to parse api key", err)
	}

	return key.Key
}

func (c *CarbonForecaster) GetZone(ctx context.Context) string {
	theLogger := withSilent(ctx, false)
	raw, err := os.ReadFile(c.GeolocationFile)
	if err != nil {
		theLogger.Error("failed to read geolocation file ", c.GeolocationFile, err)
	}
	var loc Geolocation
	if err = json.Unmarshal(raw, &loc); err != nil {
		theLogger.Error("failed to parse geolocation", err)
	}

	return loc.Zone
}

func (c *CarbonForecaster) PerformHttpReq(ctx context.Context, url string, key string) (int, []byte) {
	theLogger := withSilent(ctx, false)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-BLOBR-KEY", key)

	res, _ := http.DefaultClient.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			theLogger.Error("Could not close the body", err)
		}
	}(res.Body)

	body, _ := ioutil.ReadAll(res.Body)

	return res.StatusCode, body
}

func (c *CarbonForecaster) GetForecast(ctx context.Context, zone string, key string) (int, ElectricityMapForecastResponse) {
	theLogger := withSilent(ctx, false)
	forecastUrl := "https://api-access.electricitymaps.com/2w97h07rvxvuaa1g/carbon-intensity/forecast?zone=" + zone

	statusCode, body := c.PerformHttpReq(ctx, forecastUrl, key)

	var resp ElectricityMapForecastResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		theLogger.Error("failed to parse electricitymaps forecast response", err)
	}

	return statusCode, resp
}

func (c *CarbonForecaster) GetHistory(ctx context.Context, zone string, key string) (int, ElectricityMapHistoryResponse) {
	theLogger := withSilent(ctx, false)
	forecastUrl := "https://api-access.electricitymaps.com/2w97h07rvxvuaa1g/carbon-intensity/history?zone=" + zone

	statusCode, body := c.PerformHttpReq(ctx, forecastUrl, key)

	var resp ElectricityMapHistoryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		theLogger.Error("failed to parse electricitymaps history response", err)
	}

	return statusCode, resp
}

func (c *CarbonForecaster) GetElectricityCarbonIntensity(ctx context.Context) []uint32 {
	const (
		datetimeLayout = "2006-01-02T15:04:05.000Z"
	)

	result := make([]uint32, 24)

	zone := c.GetZone(ctx)
	key := c.GetAPIKey(ctx)

	if c.ForecastAvailable {
		forecastStatus, forecasts := c.GetForecast(ctx, zone, key)

		if forecastStatus == 401 {
			c.ForecastAvailable = false
		}

		if c.ForecastAvailable && forecasts.Zone != "" {
			c.Success = true
			counter := 0
			for _, forcast := range forecasts.Forecast {
				t, err := time.Parse(forcast.Datetime, datetimeLayout)
				if err != nil {
					fmt.Println(err)
				}

				if t.Hour() < time.Now().UTC().Hour() {
					continue
				}

				result[counter] = forcast.CarbonIntensity
				counter++
				if counter >= 24 {
					break
				}
			}
		}
	} else {
		_, histories := c.GetHistory(ctx, zone, key)
		c.Success = histories.Zone != ""
		counter := 0
		for i := len(histories.History) - 1; i >= 0; i-- {
			result[23-counter] = histories.History[i].CarbonIntensity
			counter++
			if counter >= 24 {
				break
			}
		}
	}

	return result
}

func (c *CarbonForecaster) ForecastCarbon(ctx context.Context) {
	if !c.Tick.Passed() && c.Success {
		return
	}

	if c.Tick.Passed() {
		c.Success = false
		c.ForecastAvailable = true
	}

	theLogger := withSilent(ctx, false)
	theLogger.Info("ForecastCarbon")
	staticInfoConfig := StaticInfoCfg{
		Carbon: make(map[common.IFIDType]InterfaceCarbon),
	}

	CIE := c.GetElectricityCarbonIntensity(ctx)

	if !c.Success {
		return
	}

	for _, iface1 := range c.AllInterfaces.All() {
		idx1 := iface1.TopoInfo().ID
		staticInfoConfig.Carbon[common.IFIDType(idx1)] = InterfaceCarbon{
			Intra: make(map[common.IFIDType][]uint8),
		}
		for _, iface2 := range c.AllInterfaces.All() {
			idx2 := iface2.TopoInfo().ID
			if idx2 == idx1 {
				continue
			}

			staticInfoConfig.Carbon[common.IFIDType(idx1)].Intra[common.IFIDType(idx2)] = c.ComputeCarbonForecastsBetweenInterfaces(ctx, idx1, idx2, CIE)

		}
	}

	raw, err := json.Marshal(staticInfoConfig)

	if err != nil {
		theLogger.Error("Carbon forecaster could no marshal json", "err", err)
	}

	err = os.WriteFile(c.CarbonForecastFile, raw, 0666)
	if err != nil {
		theLogger.Error("Could not write carbon forecasts to json file", "err", err, c.CarbonForecastFile)
	}
}

func (c *CarbonForecaster) ComputeCarbonForecastsBetweenInterfaces(ctx context.Context, idx1 uint16, idx2 uint16, CIE []uint32) []uint8 {
	theLogger := withSilent(ctx, false)
	forecastVector := make([]uint8, 48)
	// We do not have any information about routers (their location and their energy intensity)
	for i := 0; i < 24; i++ {
		theLogger.Info("CarbonForcaster " + strconv.FormatInt(int64(i), 10) + " " + strconv.FormatInt(int64(CIE[i]), 10))
		forecastVector[i] = uint8((100 * CIE[i]) / 3600) // we assume only one router with electricity intensity of 10 W / Gb
	}

	for i := 24; i < 48; i++ {
		forecastVector[i] = uint8(100 * CIE[i-24] / 3600)
	}

	return forecastVector
}
