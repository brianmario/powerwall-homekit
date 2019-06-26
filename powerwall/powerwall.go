package powerwall

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
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

type Powerwall struct {
	*accessory.Accessory

	battery *service.BatteryService
	ip      net.IP
}

func NewPowerwall(ip net.IP) *Powerwall {
	// TODO: get powerwall info from the from the /api/powerwalls endpoint
	info := accessory.Info{
		Name: "Powerwall",
		// Model:        "2012170-00-A",
		Manufacturer: "Tesla",
		// SerialNumber: "TG118252000S5W/TG118252000S65",
		// FirmwareRevision: "",
	}

	pw := &Powerwall{ip: ip}
	pw.Accessory = accessory.New(info, accessory.TypeOther)
	pw.battery = service.NewBatteryService()
	pw.AddService(pw.battery.Service)

	return pw
}

func (pw *Powerwall) Update() error {
	err := pw.updateChargePercentage()
	if err != nil {
		return err
	}

	return pw.updateChargingState()
}

func (pw *Powerwall) makeRequest(uri string, ret interface{}) error {
	url := fmt.Sprintf("https://%s%s", pw.ip.String(), uri)

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

type apiBatteryStatusResponse struct {
	Percentage float64 `json:"percentage"`
}

func (pw *Powerwall) updateChargePercentage() error {
	batteryStatus := &apiBatteryStatusResponse{}

	err := pw.makeRequest("/api/system_status/soe", batteryStatus)
	if err != nil {
		return err
	}

	rounded := math.RoundToEven(batteryStatus.Percentage)

	intRounded := int(rounded)

	pw.battery.BatteryLevel.SetValue(intRounded)

	if intRounded <= 5 {
		pw.battery.StatusLowBattery.SetValue(characteristic.StatusLowBatteryBatteryLevelLow)
	} else {
		pw.battery.StatusLowBattery.SetValue(characteristic.StatusLowBatteryBatteryLevelNormal)
	}

	return nil
}

type apiChargingStatusResponse struct {
	Battery struct {
		InstantPower float64 `json:"instant_power"`
	} `json:"battery"`
}

func (pw *Powerwall) updateChargingState() error {
	chargingStatus := &apiChargingStatusResponse{}

	err := pw.makeRequest("/api/meters/aggregates", chargingStatus)
	if err != nil {
		return err
	}

	// this will just be the most recent value
	charge := pw.battery.BatteryLevel.GetValue()

	if charge == 100 {
		// battery is fully charged
		pw.battery.ChargingState.SetValue(characteristic.ChargingStateNotChargeable)
	} else if chargingStatus.Battery.InstantPower < 0 {
		// battery is charging
		pw.battery.ChargingState.SetValue(characteristic.ChargingStateCharging)
	} else {
		// battery is discharging
		pw.battery.ChargingState.SetValue(characteristic.ChargingStateNotCharging)
	}

	return nil
}
