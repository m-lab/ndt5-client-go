package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bassosimone/ndt5-client-go"
	"github.com/m-lab/go/rtx"
)

func main() {
	client := ndt5.NewClient()
	out, err := client.Start(context.Background())
	rtx.Must(err, "client.Start failed")
	var extra string
	for ev := range out {
		if ev.InfoMessage != nil {
			fmt.Printf("%s%s\n", extra, strings.Trim(ev.InfoMessage.Message, "\t\n "))
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
	formatted := float64(8*speed.Count) / float64(speed.Elapsed.Microseconds())
	return fmt.Sprintf("%11.4f Mbit/s", formatted)
}
