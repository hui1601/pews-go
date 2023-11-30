# PEWS-GO
대한민국 기상청의 PEWS(사용자 맞춤형 지진정보서비스) 파서입니다.

PEWS는 서버에서 각종 정보를 바이너리 형식을 이용해 받아옵니다. 이 구조를 재구현한 파서입니다.

기본 데이터 주소는 `https://www.weather.go.kr/pews/data`을 사용합니다. 필요시 변경 후 이용바랍니다.

`func GetStationList() ([]Station, error)`을 `func GetStationData(stationLength int) (*EarthquakeMessage, error)`전에 반드시 호출하여야합니다.
