package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"time"
)

type Fronius struct {
	baseURL url.URL
	Client  *http.Client
}

func NewFronius(urlString string) (*Fronius, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	u.Path = filepath.Join(u.Path, "solar_api/v1/")
	return &Fronius{
		baseURL: *u,
		Client:  http.DefaultClient,
	}, nil
}

type StatusCode int

const (
	StatusCodeStartup     = "Startup"
	StatusCodeRunning     = "Running"
	StatusCodeStandby     = "Standby"
	StatusCodeBootloading = "Bootloading"
	StatusCodeError       = "Error"
	StatusCodeIdle        = "Idle"
	StatusCodeReady       = "Ready"
	StatusCodeSleeping    = "Sleeping"
	StatusCodeUnknown     = "Unknown"
	StatusCodeInvalid     = "INVALID"
)

func StatusCodes() []string {
	return []string{
		StatusCodeStartup,
		StatusCodeRunning,
		StatusCodeStandby,
		StatusCodeBootloading,
		StatusCodeError,
		StatusCodeIdle,
		StatusCodeReady,
		StatusCodeSleeping,
		StatusCodeUnknown,
		StatusCodeInvalid,
	}
}

func (i StatusCode) String() string {
	if i >= 0 && i < 7 {
		return StatusCodeStartup
	}
	if i == 7 {
		return StatusCodeRunning
	}
	if i == 8 {
		return StatusCodeStandby
	}
	if i == 9 {
		return StatusCodeBootloading
	}
	if i == 10 {
		return StatusCodeError
	}
	if i == 11 {
		return StatusCodeIdle
	}
	if i == 12 {
		return StatusCodeReady
	}
	if i == 13 {
		return StatusCodeSleeping
	}
	if i == 255 {
		return StatusCodeUnknown
	}
	return StatusCodeInvalid
}

type Msg struct {
	Body struct {
		Data json.RawMessage `json:"Data"`
	}
	Head struct {
		RequestArguments struct {
		} `json:"RequestArguments"`
		Status struct {
			Code        int    `json:"Code"`
			Reason      string `json:"Reason"`
			UserMessage string `json:"UserMessage"`
		} `json:"Status"`
		Timestamp time.Time `json:"Timestamp"`
	} `json:"Head"`
}

func (m *Msg) Error() error {
	if m.Head.Status.Code == 0 {
		return nil
	}

	return fmt.Errorf("fronius status code=%d: msg=%s reason=%s", m.Head.Status.Code, m.Head.Status.UserMessage, m.Head.Status.Reason)
}

func (f *Fronius) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "fronius-exporter")
	return req, nil

}

type InverterInfo struct {
	Name string `json:"-"`
	// Custom name of the inverter, assigned by the customer.
	CustomName string `json:"CustomName"`
	// Device type of the inverter.
	Dt int `json:"DT"`
	// Error code that is currently present on inverter.
	ErrorCode int `json:"ErrorCode"`
	// PV power connected to this inverter (in watts).
	// If none defined, default power for this DT is used.
	PVPower int `json:"PVPower"`
	// Whether the device shall be displayed in visualizations according
	// to customer settings. (0 do not show; 1 show)
	// visualization settings.
	Show int `json:"Show"`
	// Status code reflecting the operational state of the inverter.
	StatusCode StatusCode `json:"StatusCode"`
	// # Unique ID of the inverter (e.g. serial number).
	UniqueID string `json:"UniqueID"`
}

// List all existing inverters
// /solar_api/v1/GetInverterInfo.cgi?DeviceClass=System"
func (f *Fronius) GetInverterInfo(ctx context.Context) ([]*InverterInfo, error) {
	u := f.baseURL
	u.Path = filepath.Join(u.Path, "GetInverterInfo.cgi")
	u.RawQuery = url.Values{
		"DeviceClass": []string{"System"},
	}.Encode()
	req, err := f.newRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status: %s", resp.Status)
	}
	defer resp.Body.Close()

	var msg Msg
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("error parsing json message: %w", err)
	}
	if err := msg.Error(); err != nil {
		return nil, err
	}

	inverters := make(map[string]*InverterInfo)
	if err := json.Unmarshal(msg.Body.Data, &inverters); err != nil {
		return nil, err
	}

	result := make([]*InverterInfo, 0, len(inverters))
	for name, inverter := range inverters {
		inverter.Name = name
		inverter.CustomName = html.UnescapeString(inverter.CustomName)
		result = append(result, inverter)
	}

	// sort result by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetRealtimeInverterRealtimeData
// /solar_api/v1/GetInverterRealtimeData.cgi?scope=Device&DataCollection=CommonInverterData&DeviceId=1"
func (f *Fronius) GetRealtimeInverterRealtimeData(ctx context.Context, scope, dataCollection, deviceID string) (json.RawMessage, error) {
	u := f.baseURL
	u.Path = filepath.Join(u.Path, "GetInverterRealtimeData.cgi")
	u.RawQuery = url.Values{
		"Scope":          []string{scope},
		"DataCollection": []string{dataCollection},
		"DeviceId":       []string{deviceID},
	}.Encode()
	req, err := f.newRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status: %s", resp.Status)
	}

	var msg Msg
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("error parsing json message: %w", err)
	}
	if err := msg.Error(); err != nil {
		return nil, err
	}
	return msg.Body.Data, nil
}

type DeviceStatus struct {
	ErrorCode              int        `json:"ErrorCode"`
	LEDColor               int        `json:"LEDColor"`
	LEDState               int        `json:"LEDState"`
	MgmtTimerRemainingTime int        `json:"MgmtTimerRemainingTime"`
	StateToReset           bool       `json:"StateToReset"`
	StatusCode             StatusCode `json:"StatusCode"`
}

type InverterCommonData struct {
	DayEnergy    DataValue    `json:"DAY_ENERGY"`
	DeviceStatus DeviceStatus `json:"DeviceStatus"`
	Fac          DataValue    `json:"FAC"`
	Iac          DataValue    `json:"IAC"`
	Idc          DataValue    `json:"IDC"`
	Pac          DataValue    `json:"PAC"`
	TotalEnergy  DataValue    `json:"TOTAL_ENERGY"`
	Uac          DataValue    `json:"UAC"`
	Udc          DataValue    `json:"UDC"`
	YearEnergy   DataValue    `json:"YEAR_ENERGY"`
}

func (f *Fronius) GetInverterRealtimeCommonData(ctx context.Context, deviceID string) (*InverterCommonData, error) {
	data, err := f.GetRealtimeInverterRealtimeData(ctx, "Device", "CommonInverterData", deviceID)
	if err != nil {
		return nil, err
	}

	var commonData InverterCommonData
	if err := json.Unmarshal(data, &commonData); err != nil {
		return nil, fmt.Errorf("unable to parse inverter common data: %w", err)
	}
	return &commonData, nil
}

type DataValue struct {
	Unit  string  `json:"Unit"`
	Value float64 `json:"Value"`
}

type InverterThreePhaseData struct {
	IacL1 DataValue `json:"IAC_L1"`
	IacL2 DataValue `json:"IAC_L2"`
	IacL3 DataValue `json:"IAC_L3"`
	UacL1 DataValue `json:"UAC_L1"`
	UacL2 DataValue `json:"UAC_L2"`
	UacL3 DataValue `json:"UAC_L3"`
}

func (f *Fronius) GetInverterRealtimeThreePhaseData(ctx context.Context, deviceID string) (*InverterThreePhaseData, error) {
	data, err := f.GetRealtimeInverterRealtimeData(ctx, "Device", "3PInverterData", deviceID)
	if err != nil {
		return nil, err
	}

	var threePhaseData InverterThreePhaseData
	if err := json.Unmarshal(data, &threePhaseData); err != nil {
		return nil, fmt.Errorf("unable to parse inverter three phase data: %w", err)
	}
	return &threePhaseData, nil
}

// GetRealtimePowerFlowRealtimeData
