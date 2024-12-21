package coda56

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/msh100/modem-stats/utils"
)

type Modem struct {
	IPAddress string
	Stats     []byte
	FetchTime int64
}

func (coda56 *Modem) ClearStats() {
	coda56.Stats = nil
}

func (coda56 *Modem) Type() string {
	return utils.TypeDocsis
}

func (coda56 *Modem) apiAddress(resource string) string {
	if coda56.IPAddress == "" {
		coda56.IPAddress = "192.168.100.1"
	}
	return fmt.Sprintf("http://%s/data/%s.asp?_=%d", coda56.IPAddress, resource, time.Now().Unix())
}

// {"portId":"1","frequency":"591000000","modulation":"2","signalStrength":"6.000","snr":"38.983","dsoctets":"1 * 2e32 + 426317386","correcteds":"13","uncorrect":"27","channelId":"7"}
type dsChannel struct {
	PortID         string `json:"portId"`
	Frequency      string `json:"frequency"`
	Modulation     string `json:"modulation"`
	SignalStrength string `json:"signalStrength"`
	Snr            string `json:"snr"`
	Dsoctets       string `json:"dsoctets"`
	Correcteds     string `json:"correcteds"`
	Uncorrect      string `json:"uncorrect"`
	ChannelID      string `json:"channelId"`
}

// {"receive":"1","ffttype":"4K","Subcarr0freqFreq":" 290600000","plclock":"YES","ncplock":"YES","mdc1lock":"YES","plcpower":"5.699997","SNR":"41","dsoctets":"41345036375","correcteds":"2088221076","uncorrect":"0"}
type dsOfdmChannel struct {
	Receive          string `json:"receive"`
	FFTType          string `json:"ffttype"`
	Subcarr0freqFreq string `json:"Subcarr0freqFreq"`
	PLCLock          string `json:"plclock"`
	NCPLock          string `json:"ncplock"`
	MDC1Lock         string `json:"mdc1lock"`
	PLCPower         string `json:"plcpower"`
	SNR              string `json:"SNR"`
	DSOctets         string `json:"dsoctets"`
	Correcteds       string `json:"correcteds"`
	Uncorrect        string `json:"uncorrect"`
}

// {"portId":"1","frequency":"25900000","bandwidth":"6400000","modtype":"64QAM","scdmaMode":"ATDMA","signalStrength":"31.250","channelId":"2"}
type usChannel struct {
	PortID         string `json:"portId"`
	Frequency      string `json:"frequency"`
	Bandwidth      string `json:"bandwidth"`
	Modtype        string `json:"modtype"`
	ScdmaMode      string `json:"scdmaMode"`
	SignalStrength string `json:"signalStrength"`
	ChannelID      string `json:"channelId"`
}

// {"uschindex":"0","state":"  DISABLED","frequency":"0","digAtten":"    0.0000","digAttenBo":"    0.0000","channelBw":"    0.0000","repPower":"    0.0000","repPower1_6":"    0.0000","fftVal":"2K"}
type usOfdmChannel struct {
	USCHIndex   string `json:"uschindex"`
	State       string `json:"state"`
	Frequency   string `json:"frequency"`
	DigAtten    string `json:"digAtten"`
	DigAttenBo  string `json:"digAttenBo"`
	ChannelBw   string `json:"channelBw"`
	RepPower    string `json:"repPower"`
	RepPower1_6 string `json:"repPower1_6"`
	FFTVal      string `json:"fftVal"`
}

type resultsStruct struct {
	Downstream     []dsChannel     `json:"dsinfo"`
	DownstreamOfdm []dsOfdmChannel `json:"dsofdminfo"`
	Upstream       []usChannel     `json:"usinfo"`
	UpstreamOfdm   []usOfdmChannel `json:"usofdminfo"`
}

func (coda56 *Modem) get(resource string) string {
	var httpClient http.Client

	req, err := http.NewRequest(http.MethodGet, coda56.apiAddress(resource), nil)
	if err != nil {
		return "null"
	}
	// spoof user agent to work around bot detection
	//	req.Header["User-Agent"] = []string{"Mozilla/5.0 (X11; CrOS x86_64 8172.45.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.64 Safari/537.36"}
	req.Header["User-Agent"] = []string{"curl/8.5.0"}
	req.Header["Accept"] = []string{"*/*"}

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Get error", err)
		return "null"
	}
	data, err := ioutil.ReadAll(res.Body)
	if err == nil {
		return string(data)
	} else {
		fmt.Println("ReadAll error", err)
		return "null"
	}
}

func atoi(s string) int {
	i, _ := strconv.Atoi(strings.Trim(s, " "))
	return i
}

func atof(s string, scale float64) int {
	f, _ := strconv.ParseFloat(strings.Trim(s, " "), 64)
	return int(f * scale)
}

func (coda56 *Modem) ParseStats() (utils.ModemStats, error) {
	if coda56.Stats == nil {
		timeStart := time.Now().UnixNano() / int64(time.Millisecond)
		coda56.Stats = []byte(fmt.Sprintf(`{"dsinfo":%s, "dsofdminfo":%s, "usinfo":%s, "usofdminfo":%s}`,
			coda56.get("dsinfo"), coda56.get("dsofdminfo"), coda56.get("usinfo"), coda56.get("usofdminfo")))
		coda56.FetchTime = (time.Now().UnixNano() / int64(time.Millisecond)) - timeStart
	}

	var upChannels []utils.ModemChannel
	var downChannels []utils.ModemChannel

	var results resultsStruct
	json.Unmarshal(coda56.Stats, &results)

	for _, downstream := range results.Downstream {
		modulation := "unknown"
		switch downstream.Modulation {
		case "2":
			modulation = "256QAM"
		}

		downChannels = append(downChannels, utils.ModemChannel{
			ChannelID:  atoi(downstream.ChannelID),
			Channel:    atoi(downstream.PortID),
			Frequency:  atoi(downstream.Frequency),
			Snr:        atof(downstream.Snr, 10),
			Power:      atof(downstream.SignalStrength, 10),
			Prerserr:   atoi(downstream.Correcteds) + atoi(downstream.Uncorrect),
			Postrserr:  atoi(downstream.Uncorrect),
			Modulation: modulation,
			Scheme:     "SC-QAM",
		})
	}

	for index, downstream := range results.DownstreamOfdm {
		if downstream.PLCLock != "YES" || downstream.NCPLock != "YES" || downstream.MDC1Lock != "YES" {
			continue
		}

		downChannels = append(downChannels, utils.ModemChannel{
			Channel:    index,
			Frequency:  atoi(downstream.Subcarr0freqFreq),
			Snr:        atof(downstream.SNR, 10),
			Power:      atof(downstream.PLCPower, 10),
			Prerserr:   atoi(downstream.Correcteds) + atoi(downstream.Uncorrect),
			Postrserr:  atoi(downstream.Uncorrect),
			Modulation: downstream.FFTType,
			Scheme:     "OFDM",
		})
	}

	for _, upstream := range results.Upstream {
		upChannels = append(upChannels, utils.ModemChannel{
			ChannelID:  atoi(upstream.ChannelID),
			Channel:    atoi(upstream.PortID),
			Frequency:  atoi(upstream.Frequency),
			Power:      atof(upstream.SignalStrength, 10),
			Modulation: upstream.Modtype,
			Scheme:     upstream.ScdmaMode,
		})
	}

	// TODO: upstream OFDM

	return utils.ModemStats{
		UpChannels:   upChannels,
		DownChannels: downChannels,
		FetchTime:    coda56.FetchTime,
	}, nil
}
