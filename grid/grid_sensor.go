package grid

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

var httpClient *http.Client

func init() {
	// ignore bad SSL certificates for the powerwall :(
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient = &http.Client{
		Transport: transCfg,
		Timeout:   time.Second * 2,
	}
}

type Sensor struct {
	*accessory.Accessory

	sensor *service.ContactSensor

	ip net.IP
}

func NewSensor(ip net.IP) *Sensor {
	info := accessory.Info{Name: "Grid Power"}

	s := &Sensor{ip: ip}
	s.Accessory = accessory.New(info, accessory.TypeSensor)
	s.sensor = service.NewContactSensor()
	s.AddService(s.sensor.Service)

	s.sensor.ContactSensorState.SetValue(s.getSensorState())
	s.sensor.ContactSensorState.OnValueRemoteGet(s.getSensorState)

	return s
}

func (s *Sensor) makeRequest(uri string, ret interface{}) error {
	url := fmt.Sprintf("https://%s%s", s.ip.String(), uri)

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(ret)
	if err != nil {
		return err
	}

	return nil
}

type apiGridConnectionStatus struct {
	GridStatus string `json:"grid_status"`
}

func (s *Sensor) getSensorState() int {
	gridConnectionStatus := &apiGridConnectionStatus{}

	err := s.makeRequest("/api/system_status/grid_status", gridConnectionStatus)
	if err != nil {
		fmt.Printf("getSensorState error: %+v\n", err)

		return -1
	}

	switch gridConnectionStatus.GridStatus {
	case "SystemIslandedActive": // grid is down
		return characteristic.ContactSensorStateContactNotDetected
	case "SystemGridConnected": // grid is up
		fallthrough
	case "SystemTransitionToGrid": // grid is restored but not yet in sync
		fallthrough
	default:
		return characteristic.ContactSensorStateContactDetected
	}
}
