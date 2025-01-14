package deviceLog

import (
	"encoding/json"
	"fmt"
	"github.com/ruraomsk/TLServer/internal/model/accToken"
	"github.com/ruraomsk/TLServer/internal/model/data"
	"net/http"
	"time"

	u "github.com/ruraomsk/TLServer/internal/utils"
	"github.com/ruraomsk/TLServer/logger"
)

//DeviceLog описание таблицы, храняшей лог от устройств
type DeviceLog struct {
	Time time.Time `json:"time"` //время записи
	ID   int       `json:"id"`   //id устройства которое прислало информацию
	Text string    `json:"text"` //информация о событие
	Type int       `json:"type"` //тип сообщения
}

//LogDeviceInfo структура запроса пользователя за данными в бд
type LogDeviceInfo struct {
	Devices   []BusyArm `json:"devices"`   //информация о девайсах
	TimeStart time.Time `json:"timeStart"` //время начала отсчета
	TimeEnd   time.Time `json:"timeEnd"`   //время конца отсчета
}

//BusyArm информация о занятом перекрестке
type BusyArm struct {
	Region      string `json:"region"`      //регион
	Area        string `json:"area"`        //район
	ID          int    `json:"ID"`          //ID
	Description string `json:"description"` //описание
	Idevice     int    `json:"idevice"`     //уникальный номер устройства
}

//shortInfo короткое описание
type shortInfo struct {
	Region      string `json:"region"`      //регион
	Area        string `json:"area"`        //район
	ID          int    `json:"ID"`          //ID
	Description string `json:"description"` //описание
}

//toStr конвертировать в структуру
func (busyArm *BusyArm) toStruct(str string) (err error) {
	err = json.Unmarshal([]byte(str), busyArm)
	if err != nil {
		return err
	}
	return nil
}

//DisplayDeviceLog формирование начальной информации отображения логов устройства
func DisplayDeviceLog(accInfo *accToken.Token) u.Response {
	db, id := data.GetDB()
	defer data.FreeDB(id)
	var devices []BusyArm
	var sqlStr = fmt.Sprintf(`SELECT distinct on (crossinfo->'region', crossinfo->'area', crossinfo->'ID', id) crossinfo, id FROM public.logdevice`)
	if accInfo.Region != "*" {
		sqlStr += fmt.Sprintf(` WHERE crossinfo::jsonb @> '{"region": "%v"}'::jsonb`, accInfo.Region)
	}
	rowsDevice, err := db.Query(sqlStr)
	if err != nil {
		return u.Message(http.StatusInternalServerError, "connection to DB error. Please try again")
	}
	for rowsDevice.Next() {
		var (
			tempDev BusyArm
			infoStr string
			idevice int
		)
		err := rowsDevice.Scan(&infoStr, &idevice)
		if err != nil {
			logger.Error.Println("|Message: Incorrect data ", err.Error())
			return u.Message(http.StatusInternalServerError, "incorrect data. Please report it to Admin")
		}
		err = tempDev.toStruct(infoStr)
		tempDev.Idevice = idevice
		if err != nil {
			logger.Error.Println("|Message: Data can't convert ", err.Error())
			return u.Message(http.StatusInternalServerError, "data can't convert. Please report it to Admin")
		}
		if tempDev.ID != 0 && tempDev.Area != "0" && tempDev.Region != "0" {
			devices = append(devices, tempDev)
		}
	}
	resp := u.Message(http.StatusOK, "list of device")
	resp.Obj["devices"] = devices

	return resp
}

//DisplayDeviceLogInfo обработчик запроса пользователя, выгрузка логов за запрошенный период
func DisplayDeviceLogInfo(arms LogDeviceInfo) u.Response {
	db, id := data.GetDB()
	dbh, idh := data.GetDB()
	defer func() {
		data.FreeDB(id)
		data.FreeDB(idh)
	}()
	if len(arms.Devices) <= 0 {
		return u.Message(http.StatusBadRequest, "no one devices selected")
	}
	var mapDevice = make(map[string][]DeviceLog, 0)
	for _, arm := range arms.Devices {
		var (
			listDevicesLog []DeviceLog
			tempInfo       = shortInfo{ID: arm.ID, Area: arm.Area, Region: arm.Region, Description: arm.Description}
			rawByte, _     = json.Marshal(tempInfo) //перобразование структуру в строку для использования в ключе
		)
		mapDevice[string(rawByte)] = make([]DeviceLog, 0)
		crossInfo := fmt.Sprintf(`crossinfo::jsonb @> '{"ID": %v, "area": "%v", "region": "%v"}'::jsonb`, arm.ID, arm.Area, arm.Region)
		timeInfo := fmt.Sprintf(`tm > '%v' and tm < '%v'`, arms.TimeStart.Format("2006-01-02 15:04:05"), arms.TimeEnd.Format("2006-01-02 15:04:05"))
		sqlStr := fmt.Sprintf(`SELECT crossinfo->'type', tm, id, txt FROM public.logdevice where %v and %v
									UNION (SELECT distinct on (crossinfo->'type') crossinfo->'type', tm, id, txt  FROM public.logdevice where %v and tm <'%v' ORDER BY crossinfo->'type', tm desc)
									ORDER BY tm DESC`,
			crossInfo, timeInfo, crossInfo, arms.TimeStart.Format("2006-01-02 15:04:05"))
		rowsDevices, err := db.Query(sqlStr)
		if err != nil {
			return u.Message(http.StatusInternalServerError, "Connection to DB error. Please try again")
		}
		for rowsDevices.Next() {
			var tempDev DeviceLog
			err := rowsDevices.Scan(&tempDev.Type, &tempDev.Time, &tempDev.ID, &tempDev.Text)
			if err != nil {
				logger.Error.Println("|Message: Incorrect data ", err.Error())
				return u.Message(http.StatusInternalServerError, "incorrect data. Please report it to Admin")
			}
			//tempDev.Devices = arm
			listDevicesLog = append(listDevicesLog, tempDev)
		}
		sqlStrH := fmt.Sprintf(`SELECT crossinfo->'type', tm, id, txt FROM public.loghistory where %v and %v
									UNION (SELECT distinct on (crossinfo->'type') crossinfo->'type', tm, id, txt  FROM public.logdevice where %v and tm <'%v' ORDER BY crossinfo->'type', tm desc)
									ORDER BY tm DESC`,
			crossInfo, timeInfo, crossInfo, arms.TimeStart.Format("2006-01-02 15:04:05"))
		rowsDevicesH, err := dbh.Query(sqlStrH)
		if err != nil {
			return u.Message(http.StatusInternalServerError, "Connection to DB error. Please try again")
		}
		for rowsDevicesH.Next() {
			var tempDev DeviceLog
			err := rowsDevicesH.Scan(&tempDev.Type, &tempDev.Time, &tempDev.ID, &tempDev.Text)
			if err != nil {
				logger.Error.Println("|Message: Incorrect data ", err.Error())
				return u.Message(http.StatusInternalServerError, "incorrect data. Please report it to Admin")
			}
			//tempDev.Devices = arm
			listDevicesLog = append(listDevicesLog, tempDev)
		}

		mapDevice[string(rawByte)] = listDevicesLog
	}

	resp := u.Message(http.StatusOK, "get device Log")
	resp.Obj["deviceLogs"] = mapDevice
	return resp
}
