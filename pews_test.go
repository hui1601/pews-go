package pews_test

import (
	pews "go-pews"
	"testing"
	"time"
)

func TestGetStationList(t *testing.T) {
	stations, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stations) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stations))
	}
}

func TestGetStationData(t *testing.T) {
	stations, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stations) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stations))
	}
	data, err := pews.GetStationData(len(stations))
	if err != nil {
		t.Error(err)
	}

	if len(data.MMI) != len(stations) {
		t.Errorf("mmi length is not matched with stations length.\n:Result: %d", len(data.MMI))
	}

	if data.LastEarthquakeId == "" {
		t.Errorf("last earthquake id is empty.")
	}

	if data.Phase == 0 {
		t.Errorf("status is not set.")
	}
}

func parseKMATimeString(timeString string) time.Time {
	t, _ := time.Parse("20060102150405", timeString)
	loc, _ := time.LoadLocation("Asia/Seoul")
	return t.In(loc)
}

func TestStartSimulation(t *testing.T) {
	simulationData := new(pews.SimulationData)

	// 2021 Jeju earthquake
	simulationDuration := 7 * time.Minute
	simulationId := "2021007178"
	simulationStartTime := parseKMATimeString("20211214081905")
	earthquakeAlertTime := parseKMATimeString("20211214081931")
	earthquakeInfoTime := parseKMATimeString("20211214082327")

	simulationData.StartTime = simulationStartTime
	simulationData.Duration = simulationDuration
	simulationData.EarthquakeId = simulationId

	pews.StartSimulation(*simulationData)
	testStartTime := time.Now()
	stationList, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stationList) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stationList))
	}
	for {
		currentTime := simulationStartTime.Add(time.Duration(time.Now().Unix()-testStartTime.Unix()) * time.Second)
		message, err := pews.GetStationData(len(stationList))
		if err != nil {
			t.Error(err)
		}
		if testStartTime.Add(simulationDuration).Before(time.Now()) {
			break
		}
		t.Logf("### %s ###\nPhase: P%d", currentTime.String(), message.Phase)
		if currentTime.Before(earthquakeAlertTime) && message.Phase != pews.PhaseNormal {
			t.Errorf("status is not set to 1(Normal).\n:Result: %d", message.Phase)
		} else if currentTime.After(earthquakeAlertTime) && currentTime.Before(earthquakeInfoTime) && message.Phase != pews.PhaseAlert {
			t.Errorf("status is not set to 2(Alert).\n:Result: %d", message.Phase)
		} else if currentTime.After(earthquakeInfoTime) && message.Phase != pews.PhaseInfo {
			t.Errorf("status is not set to 3(Info).\n:Result: %d", message.Phase)
		}
		time.Sleep(time.Millisecond * 800)
	}
}
