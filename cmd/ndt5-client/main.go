package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/bassosimone/ndt5-client-go"
	"github.com/m-lab/go/rtx"
)

var (
	flagHostname = flag.String("hostname", "", "Measurement server hostname")
)

func main() {
	flag.Parse()
	client := ndt5.NewClient()
	client.FQDN = *flagHostname
	out, err := client.Start(context.Background())
	rtx.Must(err, "client.Start failed")
	var extra string
	for ev := range out {
		if ev.InfoMessage != nil {
			fmt.Printf("%s%s\n", extra, strings.Trim(ev.InfoMessage.Message, "\t\n "))
			extra = ""
		}
		if ev.WarningMessage != nil {
			fmt.Printf("%swarning: %s\n", extra, ev.WarningMessage.Error.Error())
			extra = ""
		}
		if ev.ErrorMessage != nil {
			fmt.Printf("%serror: %s\n", extra, ev.ErrorMessage.Error.Error())
			extra = ""
		}
		if ev.CurDownloadSpeed != nil {
			fmt.Printf("\rdownload: %s", computeSpeed(ev.CurDownloadSpeed))
			extra = "\n"
		}
		if ev.CurUploadSpeed != nil {
			fmt.Printf("\rupload:   %s", computeSpeed(ev.CurUploadSpeed))
			extra = "\n"
		}
	}
}

func computeSpeed(speed *ndt5.Speed) string {
	elapsed := speed.Elapsed.Seconds() * 1e06
	formatted := float64(8*speed.Count) / elapsed
	return fmt.Sprintf("%11.4f Mbit/s", formatted)
}
