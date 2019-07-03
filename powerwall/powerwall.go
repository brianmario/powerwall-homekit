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

	pw.battery.BatteryLevel.OnValueRemoteGet(pw.getChargePercentage)
	pw.battery.ChargingState.OnValueRemoteGet(pw.getChargingState)
	pw.battery.StatusLowBattery.OnValueRemoteGet(pw.getLowBatteryStatus)

	return pw
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

func (pw *Powerwall) getChargePercentage() int {
	batteryStatus := &apiBatteryStatusResponse{}

	err := pw.makeRequest("/api/system_status/soe", batteryStatus)
	if err != nil {
		fmt.Printf("updateChargePercentage error: %+v\n", err)

		return -1
	}

	rounded := math.RoundToEven(batteryStatus.Percentage)

	return int(rounded)
}

type apiChargingStatusResponse struct {
	Battery struct {
		InstantPower float64 `json:"instant_power"`
	} `json:"battery"`
}

func (pw *Powerwall) getChargingState() int {
	chargingStatus := &apiChargingStatusResponse{}

	err := pw.makeRequest("/api/meters/aggregates", chargingStatus)
	if err != nil {
		fmt.Printf("updateChargingState error: %+v\n", err)

		return -1
	}

	charge := pw.battery.BatteryLevel.GetValue()

	if charge == 100 {
		// battery is fully charged
		return characteristic.ChargingStateNotChargeable
	} else if chargingStatus.Battery.InstantPower < 0 {
		// battery is charging
		return characteristic.ChargingStateCharging
	}

	// battery is discharging
	return characteristic.ChargingStateNotCharging
}

func (pw *Powerwall) getLowBatteryStatus() int {
	charge := pw.battery.BatteryLevel.GetValue()

	if charge <= 5 {
		return characteristic.StatusLowBatteryBatteryLevelLow
	}

	return characteristic.StatusLowBatteryBatteryLevelNormal
}
