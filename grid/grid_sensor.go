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

	return s
}

func (s *Sensor) Update() error {
	return s.updateGridConnectionStatus()
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

func (s *Sensor) updateGridConnectionStatus() error {
	gridConnectionStatus := &apiGridConnectionStatus{}

	err := s.makeRequest("/api/system_status/grid_status", gridConnectionStatus)
	if err != nil {
		return err
	}

	switch gridConnectionStatus.GridStatus {
	case "SystemGridConnected": // grid is up
		s.sensor.ContactSensorState.SetValue(characteristic.ContactSensorStateContactDetected)
	case "SystemIslandedActive": // grid is down
		s.sensor.ContactSensorState.SetValue(characteristic.ContactSensorStateContactNotDetected)
	case "SystemTransitionToGrid": // grid is restored but not yet in sync
		s.sensor.ContactSensorState.SetValue(characteristic.ContactSensorStateContactDetected)
	}

	return nil
}
