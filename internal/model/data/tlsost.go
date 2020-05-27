package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JanFant/newTLServer/internal/model/locations"
	"github.com/JanFant/newTLServer/internal/model/logger"
	u "github.com/JanFant/newTLServer/internal/utils"
	agS_pudge "github.com/ruraomsk/ag-server/pudge"
	"net/http"
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

//GetLightsFromBD определяем область отображения светофоров
func GetLightsFromBD(box locations.BoxPoint) (tfdata []TrafficLights) {
	var tflight = []TrafficLights{}
	if (box.Point1.X > -180 && box.Point1.X < 0) && (box.Point0.X > 0 && box.Point0.X < 180) {
		var (
			point0 locations.Point
			point1 locations.Point
		)
		//для первую область
		point0 = box.Point0
		point1.Y = box.Point1.Y
		point1.X = 179.9999999999
		tflight = SelectTL(point0, point1, false)
		//для второй области
		point0.Y = box.Point0.Y
		point0.X = -179.9999999999
		point1 = box.Point1
		tempTF := SelectTL(point0, point1, false)
		tflight = append(tflight, tempTF...)

	} else if int(box.Point0.X) == int(box.Point1.X) {
		tflight = SelectTL(box.Point0, box.Point1, true)
	} else {
		tflight = SelectTL(box.Point0, box.Point1, false)
	}

	return tflight
}

//SelectTL возвращает массив в котором содержатся светофоры, которые попали в указанную область
func SelectTL(point0 locations.Point, point1 locations.Point, equalPoint bool) (tfdata []TrafficLights) {
	var (
		dgis     string
		sqlStr   string
		StateStr string
	)

	temp := &TrafficLights{}
	if equalPoint {
		sqlStr = fmt.Sprintf(`SELECT region, area, subarea, id, idevice, dgis, describ, state FROM public.cross`)
	} else {
		sqlStr = fmt.Sprintf("SELECT region, area, subarea, id, idevice, dgis, describ, state FROM public.cross WHERE box '((%3.15f,%3.15f),(%3.15f,%3.15f))'@> dgis", point0.Y, point0.X, point1.Y, point1.X)
	}
	rowsTL, _ := GetDB().Query(sqlStr)
	for rowsTL.Next() {
		err := rowsTL.Scan(&temp.Region.Num, &temp.Area.Num, &temp.Subarea, &temp.ID, &temp.Idevice, &dgis, &temp.Description, &StateStr)
		if err != nil {
			logger.Error.Println("|Message: No result at these points", err.Error())
			return nil
		}
		temp.Points.StrToFloat(dgis)
		//Состояние светофора!
		rState, err := ConvertStateStrToStruct(StateStr)
		if err != nil {
			logger.Error.Println("|Message: Failed to parse cross information", err.Error())
			return nil
		}
		CacheInfo.Mux.Lock()
		temp.Region.NameRegion = CacheInfo.MapRegion[temp.Region.Num]
		temp.Area.NameArea = CacheInfo.MapArea[temp.Region.NameRegion][temp.Area.Num]
		temp.Sost.Description = CacheInfo.MapTLSost[temp.Sost.Num]
		CacheInfo.Mux.Unlock()
		temp.Sost.Num = rState.StatusDevice
		tfdata = append(tfdata, *temp)
	}

	return tfdata
}

//GetAllTrafficLights запрос информации об всех сфетофорах из БД
func GetAllTrafficLights() (tfData []TrafficLights) {
	var dgis string
	temp := &TrafficLights{}
	sqlStr := fmt.Sprintf("SELECT region, id, area, dgis, describ FROM public.cross")
	rows, _ := GetDB().Query(sqlStr)
	for rows.Next() {
		_ = rows.Scan(&temp.Region.Num, &temp.ID, &temp.Area.Num, &dgis, &temp.Description)
		temp.Points.StrToFloat(dgis)
		tfData = append(tfData, *temp)
	}
	return
}

//ConvertStateStrToStruct разбор данных (Cross) полученных из БД в нужную структуру
func ConvertStateStrToStruct(str string) (rState agS_pudge.Cross, err error) {
	if err := json.Unmarshal([]byte(str), &rState); err != nil {
		return rState, err
	}
	return rState, nil
}

//ConvertDevStrToStruct разбор данных (Controller) полученных из БД в нужную структуру
func ConvertDevStrToStruct(str string) (controller agS_pudge.Controller, err error) {
	if err := json.Unmarshal([]byte(str), &controller); err != nil {
		return controller, err
	}
	return controller, nil
}

//GetCrossInfo сбор информации для пользователя о выбранном перекрестке
func GetCrossInfo(TLignt TrafficLights) u.Response {
	var (
		dgis     string
		sqlStr   string
		stateStr string
	)

	sqlStr = fmt.Sprintf("SELECT area, subarea, idevice, dgis, describ, state FROM public.cross WHERE region = %v and id = %v and area = %v", TLignt.Region.Num, TLignt.ID, TLignt.Area.Num)
	rowsTL := GetDB().QueryRow(sqlStr)
	err := rowsTL.Scan(&TLignt.Area.Num, &TLignt.Subarea, &TLignt.Idevice, &dgis, &TLignt.Description, &stateStr)
	if err != nil {
		logger.Error.Println("|Message: No result at these points, table cross", err.Error())
		return u.Message(http.StatusInternalServerError, "no result at these points")
	}
	TLignt.Points.StrToFloat(dgis)
	//Состояние светофора!
	rState, err := ConvertStateStrToStruct(stateStr)
	if err != nil {
		logger.Error.Println("|Message: Failed to parse cross information", err.Error())
		return u.Message(http.StatusInternalServerError, "failed to parse cross information")
	}

	resp := u.Message(http.StatusOK, "cross information")

	CacheInfo.Mux.Lock()
	TLignt.Region.NameRegion = CacheInfo.MapRegion[TLignt.Region.Num]
	TLignt.Area.NameArea = CacheInfo.MapArea[TLignt.Region.NameRegion][TLignt.Area.Num]
	TLignt.Sost.Num = rState.StatusDevice
	TLignt.Sost.Description = CacheInfo.MapTLSost[TLignt.Sost.Num]
	CacheInfo.Mux.Unlock()
	resp.Obj["DontWrite"] = "true"
	resp.Obj["cross"] = TLignt
	resp.Obj["state"] = rState
	return resp
}

//GetCrossDevInfo сбор информации для пользователя о выбранном перекрестке (информацию о девайсе)
func GetCrossDevInfo(idevice string) u.Response {
	var (
		sqlStr string
		devStr string
	)
	resp := u.Message(http.StatusOK, "cross information")
	sqlStr = fmt.Sprintf(`SELECT device FROM public.devices WHERE id = %v`, idevice)
	err := GetDB().QueryRow(sqlStr).Scan(&devStr)
	if err != nil {
		logger.Error.Println("|Message: No result at these points, table device", err.Error())
		return u.Message(http.StatusBadRequest, "no result at these points")
	} else {
		device, err := ConvertDevStrToStruct(devStr)
		if err != nil {
			logger.Error.Println("|Message: Failed to parse cross information", err.Error())
			return u.Message(http.StatusInternalServerError, "failed to parse cross information")
		}
		resp.Obj["device"] = device
	}
	resp.Obj["DontWrite"] = "true"
	return resp
}

//MakeBoxPoint расчет координат для перемещения по карте
func (location *Locations) MakeBoxPoint() (box locations.BoxPoint, err error) {
	var sqlStr = `SELECT Min(dgis[0]) as "Y0", Min(convTo360(dgis[1])) as "X0", Max(dgis[0]) as "Y1", Max(convTo360(dgis[1])) as "X1"  FROM public."cross"`
	tempStr := " WHERE "
	tempStr += fmt.Sprintf("region = %v AND area in (", location.Region)
	for numArea, area := range location.Area {
		if numArea == 0 {
			tempStr += fmt.Sprintf("%v", area)
		} else {
			tempStr += fmt.Sprintf(",%v", area)
		}
	}
	tempStr += ")"
	sqlStr += tempStr
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