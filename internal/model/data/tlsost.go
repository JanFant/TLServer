package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JanFant/TLServer/internal/model/locations"
	"github.com/JanFant/TLServer/logger"
	agspudge "github.com/ruraomsk/ag-server/pudge"
)

//TrafficLights информация о светофоре
type TrafficLights struct {
	ID          int             `json:"ID"`          //Уникальный ID светофора
	Region      RegionInfo      `json:"region"`      //Регион
	Area        AreaInfo        `json:"area"`        //Район
	Subarea     int             `json:"subarea"`     //ПодРайон
	Idevice     int             `json:"idevice"`     //Реальный номер устройства
	Sost        TLSostInfo      `json:"tlsost"`      //Состояние светофора
	Description string          `json:"description"` //Описание светофора
	Points      locations.Point `json:"points"`      //Координата где находится светофор
}

//Locations информация о запрашиваемом регионе и районе карты
type Locations struct {
	Region string   `json:"region"` //регион
	Area   []string `json:"area"`   //районы
}

//SelectTL возвращает массив в котором содержатся светофоры, которые попали в указанную область
func SelectTL() (tfdata []TrafficLights) {
	var dgis string
	temp := &TrafficLights{}
	rowsTL, err := GetDB().Query(`SELECT region, area, subarea, id, idevice, dgis, describ, status FROM public.cross`)
	if err != nil {
		logger.Error.Println("|Message: db not respond", err.Error())
		return nil
	}
	for rowsTL.Next() {
		err := rowsTL.Scan(&temp.Region.Num, &temp.Area.Num, &temp.Subarea, &temp.ID, &temp.Idevice, &dgis, &temp.Description, &temp.Sost.Num)
		if err != nil {
			logger.Error.Println("|Message: No result at these points", err.Error())
			return nil
		}
		temp.Points.StrToFloat(dgis)
		CacheInfo.Mux.Lock()
		temp.Region.NameRegion = CacheInfo.MapRegion[temp.Region.Num]
		temp.Area.NameArea = CacheInfo.MapArea[temp.Region.NameRegion][temp.Area.Num]
		temp.Sost.Description = CacheInfo.MapTLSost[temp.Sost.Num]
		CacheInfo.Mux.Unlock()
		tfdata = append(tfdata, *temp)
	}

	return tfdata
}

func MapOpenInfo() (obj map[string]interface{}) {
	obj = make(map[string]interface{})

	location := &Locations{}
	box, _ := location.MakeBoxPoint()
	obj["boxPoint"] = &box
	obj["tflight"] = SelectTL()
	obj["authorizedFlag"] = false

	//собираю в кучу регионы для отображения
	chosenRegion := make(map[string]string)
	CacheInfo.Mux.Lock()
	for first, second := range CacheInfo.MapRegion {
		chosenRegion[first] = second
	}
	delete(chosenRegion, "*")
	obj["regionInfo"] = chosenRegion

	//собираю в кучу районы для отображения
	chosenArea := make(map[string]map[string]string)
	for first, second := range CacheInfo.MapArea {
		chosenArea[first] = make(map[string]string)
		chosenArea[first] = second
	}
	delete(chosenArea, "Все регионы")
	CacheInfo.Mux.Unlock()
	obj["areaInfo"] = chosenArea
	return
}

//GetAllTrafficLights запрос информации об всех сфетофорах из БД
func GetAllTrafficLights() (tfData []TrafficLights) {
	var dgis string
	temp := &TrafficLights{}
	sqlStr := fmt.Sprintf("SELECT region, id, area, dgis, describ FROM public.cross")
	rows, err := GetDB().Query(sqlStr)
	if err != nil {
		logger.Error.Println("|Message: db not respond", err.Error())
		return nil
	}
	for rows.Next() {
		_ = rows.Scan(&temp.Region.Num, &temp.ID, &temp.Area.Num, &dgis, &temp.Description)
		temp.Points.StrToFloat(dgis)
		tfData = append(tfData, *temp)
	}
	return
}

//ConvertStateStrToStruct разбор данных (Cross) полученных из БД в нужную структуру
func ConvertStateStrToStruct(str string) (rState agspudge.Cross, err error) {
	if err := json.Unmarshal([]byte(str), &rState); err != nil {
		return rState, err
	}
	return rState, nil
}

//MakeBoxPoint расчет координат для перемещения по карте
func (l *Locations) MakeBoxPoint() (box locations.BoxPoint, err error) {
	var sqlStr = `SELECT Min(dgis[0]) as "Y0", Min(convTo360(dgis[1])) as "X0", Max(dgis[0]) as "Y1", Max(convTo360(dgis[1])) as "X1"  FROM public."cross"`
	if l.Region != "" {
		tempStr := " WHERE "
		tempStr += fmt.Sprintf("region = %v AND area in (", l.Region)
		for numArea, area := range l.Area {
			if numArea == 0 {
				tempStr += fmt.Sprintf("%v", area)
			} else {
				tempStr += fmt.Sprintf(",%v", area)
			}
		}
		tempStr += ")"
		sqlStr += tempStr
	}
	row := GetDB().QueryRow(sqlStr)
	err = row.Scan(&box.Point0.Y, &box.Point0.X, &box.Point1.Y, &box.Point1.X)
	if err != nil {
		return box, errors.New(fmt.Sprintf("parserPoints. Request error: %s", err.Error()))
	}
	if box.Point0.X > 180 {
		box.Point0.X -= 360
	}
	if box.Point1.X > 180 {
		box.Point1.X -= 360
	}
	return
}
