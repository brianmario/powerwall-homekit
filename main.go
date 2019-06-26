package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/brutella/hc/accessory"

	"github.com/brianmario/powerwall-homekit/grid"
	"github.com/brianmario/powerwall-homekit/powerwall"
	"github.com/brutella/hc"
)

var powerwallIP string

const ipDefault = ""

func main() {
	flag.StringVar(&powerwallIP, "ip", ipDefault, "ip address of powerwall")

	flag.Parse()

	if powerwallIP == ipDefault {
		fmt.Printf("Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()

		os.Exit(1)
	}

	ip := net.ParseIP(powerwallIP)

	bridgeInfo := accessory.Info{Name: "Tesla Bridge"}

	bridge := accessory.NewBridge(bridgeInfo)

	pw := powerwall.NewPowerwall(ip)

	sensor := grid.NewSensor(ip)

	// TODO: pass this from the cmdline?
	pwConfig := hc.Config{Pin: "00102003"}

	// NOTE: the first accessory in the list acts as the bridge, while the rest will be linked to it
	t, err := hc.NewIPTransport(pwConfig, bridge.Accessory, pw.Accessory, sensor.Accessory)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	go func() {
		// At a minimum we'll wait 1s between loops
		//
		// Note that any of these calls may error out. Most likely these will be timeouts
		// since the Powerwall drops off wifi (well, mine does) every once in a while.
		for {
			err := pw.Update()
			if err != nil {
				fmt.Printf("powerwall update error %+v\n", err)
			}

			err = sensor.Update()
			if err != nil {
				fmt.Printf("grid sensor update error %+v\n", err)
			}

			time.Sleep(time.Second * 2)
		}
	}()

	t.Start()
}
