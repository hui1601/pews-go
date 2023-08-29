package pews

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Station struct {
	// divide by 100 before use
	Longitude int `json:"longitude"`
	// divide by 100 before use
	Latitude int `json:"latitude"`
}

type EarthquakeMessage struct {
	StationUpdateNeeded bool   `json:"stationUpdateNeeded"`
	Phase               int8   `json:"phase"`
	LastEarthquakeId    string `json:"lastEarthquakeId"`
	MMI                 []int8 `json:"mmi"`
	EarthquakeInfo      struct {
		// divide by 100 before use
		Longitude int `json:"longitude"`
		// divide by 100 before use
		Latitude         int      `json:"latitude"`
		EarthquakeId     string   `json:"earthquakeId"`
		Magnitude        int8     `json:"magnitude"`
		Depth            int8     `json:"depth"`
		Time             string   `json:"time"`
		MaxIntensity     int8     `json:"maxIntensity"`
		MaxIntensityArea []string `json:"maxIntensityArea"`
		Epicenter        string   `json:"epicenter"`
	} `json:"earthquakeInfo"`
}

type SimulationData struct {
	StartTime    time.Time // when simulation data starts(ex. 20211214081904)
	EarthquakeId string
	Duration     time.Duration
	callTime     time.Time // when simulation started(when StartStimulation called, ex. 20230214081904)
}

const (
	PhaseNormal = iota + 1
	PhaseAlert
	PhaseInfo
	PhaseUpdateInfo
)

var simulation *SimulationData

func byteToBinaryString(b byte) string {
	// convert using bit shifting
	var binaryString string
	for i := 0; i < 8; i++ {
		binaryString += strconv.Itoa(int(b >> uint(7-i) & 1))
	}
	return binaryString
}

func byteArrayToBinaryString(byteArray []byte) string {
	var binaryString string
	for _, b := range byteArray {
		binaryString += byteToBinaryString(b)
	}
	return binaryString
}

func binaryStringToInt(binaryString string) int {
	var result int
	for i := 0; i < len(binaryString); i++ {
		result += int(binaryString[i]-'0') << uint(len(binaryString)-i-1)
	}
	return result
}

func kmaTimeString() string {
	if simulation != nil {
		timeDiff := int(time.Now().Unix() - simulation.callTime.Unix())
		if timeDiff > int(simulation.Duration.Seconds()) {
			simulation = nil
		} else {
			return simulation.StartTime.Add(time.Duration(timeDiff-1) * time.Second).UTC().Format("20060102150405")
		}
	}
	return time.Now().UTC().Add(-1 * time.Second).Format("20060102150405")
}

func parseStationDataHeader(headerString string, message *EarthquakeMessage) {
	message.StationUpdateNeeded = headerString[0] == '1'
	message.Phase = (func(code string) int8 {
		// it's very awful
		// why KMA does this?!!
		switch code {
		case "00":
			return PhaseNormal
		case "01":
			return PhaseUpdateInfo
		case "10":
			return PhaseAlert
		case "11":
			return PhaseInfo
		}
		return PhaseNormal
	})(headerString[1:3])
	// data header is short when simulation.
	if simulation == nil {
		message.LastEarthquakeId = "20" + strconv.Itoa(binaryStringToInt(headerString[6:32]))
	}
}

func parseStationDataBody(bodyString string, stationLength int, message *EarthquakeMessage) {
	for i := 0; i < stationLength; i++ {
		mmiConvertArray := []int8{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 1, 1, 1}
		message.MMI = append(message.MMI, mmiConvertArray[binaryStringToInt(bodyString[i*4:i*4+4])])
	}
	if message.Phase == PhaseAlert || message.Phase == PhaseInfo {
		// very disgusting
		var earthquakeInfoBinaryString = bodyString[len(bodyString)-600:]
		message.EarthquakeInfo.Latitude = 3000 + binaryStringToInt(earthquakeInfoBinaryString[0:10])
		message.EarthquakeInfo.Longitude = 12000 + binaryStringToInt(earthquakeInfoBinaryString[10:20])
		message.EarthquakeInfo.Magnitude = int8(binaryStringToInt(earthquakeInfoBinaryString[20:27]))
		message.EarthquakeInfo.Depth = int8(binaryStringToInt(earthquakeInfoBinaryString[27:36]))
		message.EarthquakeInfo.Time = strconv.Itoa(binaryStringToInt(earthquakeInfoBinaryString[36:69])+32400) + "000"
		message.EarthquakeInfo.EarthquakeId = "20" + strconv.Itoa(binaryStringToInt(earthquakeInfoBinaryString[69:95]))
		message.EarthquakeInfo.MaxIntensity = int8(binaryStringToInt(earthquakeInfoBinaryString[95:99]))
		// It is impossible to observe maximum seismic intensity in all regions (except for the end of the earth).
		// If the maximum intensity is "I", it should be treated as an exception by KMA.
		if earthquakeInfoBinaryString[99:116] == "11111111111111111" {
			message.EarthquakeInfo.MaxIntensityArea = []string{}
		} else {
			// This order should never be changed!
			areaNames := []string{"서울", "부산", "대구", "인천", "광주", "대전", "울산", "세종", "경기", "강원", "충북", "충남", "전북", "전남", "경북", "경남", "제주"}
			for i := 99; i < 116; i++ {
				if earthquakeInfoBinaryString[i] == '1' {
					message.EarthquakeInfo.MaxIntensityArea = append(message.EarthquakeInfo.MaxIntensityArea, areaNames[i-99])
				}
			}
		}
	}
}

func GetStationList() ([]Station, error) {
	var stations []Station
	url := "https://www.weather.go.kr/pews/data/" + kmaTimeString() + ".s"
	if simulation != nil {
		url = fmt.Sprintf("https://www.weather.go.kr/pews/data/%s/%s.s", simulation.EarthquakeId, kmaTimeString())
	}
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// bodyBytes to binary string
		binaryString := byteArrayToBinaryString(bodyBytes)
		for i := 0; i < len(binaryString)/20*20; i += 20 {
			latitude := binaryStringToInt(binaryString[i : i+10])
			longitude := binaryStringToInt(binaryString[i+10 : i+20])
			stations = append(stations, Station{Longitude: 12000 + longitude, Latitude: 3000 + latitude})
		}
	}
	return stations, nil
}

func GetStationData(stationLength int) (*EarthquakeMessage, error) {
	var message EarthquakeMessage
	var client http.Client
	headerSize := 32
	url := "https://www.weather.go.kr/pews/data/" + kmaTimeString() + ".b"
	if simulation != nil {
		url = "https://www.weather.go.kr/pews/data/" + simulation.EarthquakeId + "/" + kmaTimeString() + ".b"
		headerSize = 8
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// bodyBytes to binary string
		binaryString := byteArrayToBinaryString(bodyBytes)
		header := binaryString[:headerSize]
		body := binaryString[headerSize:]
		parseStationDataHeader(header, &message)
		parseStationDataBody(body, stationLength, &message)
		if message.Phase == PhaseAlert || message.Phase == PhaseInfo {
			message.EarthquakeInfo.Epicenter = strings.Trim(string(bodyBytes[len(bodyBytes)-60:]), "\x00\x20")
		}
	}
	return &message, nil
}

func StartSimulation(data SimulationData) {
	simulation = &data
	simulation.callTime = time.Now()
}
