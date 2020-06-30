package techArm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"strconv"
	"time"
)

var connectedUsersTechArm map[*websocket.Conn]ArmInfo
var writeArm chan armResponse
var TArmNewCrossData chan bool

const devUpdate = time.Second * 1

//ArmTechReader обработчик открытия сокета для тех арм
func ArmTechReader(conn *websocket.Conn, reg int, area []string, db *sqlx.DB) {
	var armInfo = ArmInfo{Region: reg, Area: area}
	connectedUsersTechArm[conn] = armInfo
	//сформировать список перекрестков которые необходимы пользователю
	{
		var tempCrosses []CrossInfo
		crosses := getCross(armInfo.Region, db)
		for _, cross := range crosses {
			for _, area := range armInfo.Area {
				tArea, _ := strconv.Atoi(area)
				if cross.Area == tArea {
					tempCrosses = append(tempCrosses, cross)
				}
			}
		}
		resp := newArmMess(typeArmInfo, conn, nil)
		resp.Data[typeCrosses] = tempCrosses

		var tempDevises []DevInfo
		devices := getDevice(db)
		for _, dev := range devices {
			for _, area := range armInfo.Area {
				tArea, _ := strconv.Atoi(area)
				if dev.Area == tArea && dev.Region == armInfo.Region {
					tempDevises = append(tempDevises, dev)
				}
			}
		}
		resp.Data[typeDevices] = tempDevises
		resp.send()
	}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			//закрытие коннекта
			resp := newArmMess(typeClose, conn, nil)
			resp.send()
			return
		}

		typeSelect, err := setTypeMessage(p)
		if err != nil {
			resp := newArmMess(typeError, conn, nil)
			resp.Data["message"] = ErrorMessage{Error: errUnregisteredMessageType}
			resp.send()
		}
		switch typeSelect {
		case "nothing":
			{

			}
		}
	}
}

//ArmTechBroadcast передатчик для тех арм (techArm)
func ArmTechBroadcast(db *sqlx.DB) {
	connectedUsersTechArm = make(map[*websocket.Conn]ArmInfo)
	writeArm = make(chan armResponse)
	TArmNewCrossData = make(chan bool)

	readTick := time.NewTicker(devUpdate)
	defer readTick.Stop()
	var oldDevice = getDevice(db)
	for {
		select {
		case <-readTick.C:
			{
				if len(connectedUsersTechArm) > 0 {
					newDevice := getDevice(db)
					var tempDev []DevInfo
					for _, nDev := range newDevice {
						for _, oDev := range oldDevice {
							if oDev.Idevice == nDev.Idevice {
								if oDev.Device.CK != nDev.Device.CK {
									tempDev = append(tempDev, nDev)
									break
								}
							}
						}
					}
					oldDevice = newDevice
					if len(tempDev) > 0 {
						for conn, arm := range connectedUsersTechArm {
							var tDev []DevInfo
							for _, area := range arm.Area {
								tArea, _ := strconv.Atoi(area)
								for _, dev := range tempDev {
									if dev.Area == tArea && dev.Region == arm.Region {
										tDev = append(tDev, dev)
									}
								}
							}
							if len(tDev) > 0 {
								resp := newArmMess(typeDevices, conn, nil)
								resp.Data[typeDevices] = tDev
								_ = conn.WriteJSON(resp)
							}
						}
					}
				}
			}
		case <-TArmNewCrossData:
			{
				crosses := getCross(-1, db)
				for conn, arm := range connectedUsersTechArm {
					var tempCrosses []CrossInfo
					for _, area := range arm.Area {
						tArea, _ := strconv.Atoi(area)
						for _, cross := range crosses {
							if cross.Region == arm.Region && cross.Area == tArea {
								tempCrosses = append(tempCrosses, cross)
							}
						}
					}
					resp := newArmMess(typeCrosses, conn, nil)
					resp.Data[typeCrosses] = tempCrosses
					_ = resp.conn.WriteJSON(resp)
				}
			}
		case msg := <-writeArm:
			switch msg.Type {
			case typeClose:
				{
					delete(connectedUsersTechArm, msg.conn)
				}
			default:
				{
					_ = msg.conn.WriteJSON(msg)
				}
			}
		}
	}
}

func getCross(reg int, db *sqlx.DB) []CrossInfo {
	var (
		temp    CrossInfo
		crosses []CrossInfo
		rows    *sql.Rows
		sqlStr  = `SELECT region,
 					area, 
 					id,
  					idevice, 
  					describ, 
  					subarea, 
  					state->'arrays'->'type' 
  					FROM public.cross`
	)
	if reg != -1 {
		sqlStr += fmt.Sprintf(` WHERE region = %v`, reg)
	}
	rows, _ = db.Query(sqlStr)
	for rows.Next() {
		_ = rows.Scan(&temp.Region, &temp.Area, &temp.ID, &temp.Idevice, &temp.Describe, &temp.Subarea, &temp.ArrayType)
		crosses = append(crosses, temp)
	}
	return crosses
}

func getDevice(db *sqlx.DB) []DevInfo {
	var (
		temp    DevInfo
		devices []DevInfo
		dStr    string
	)
	rows, _ := db.Query(`SELECT c.region, c.area, c.idevice, d.device FROM public.cross as c, public.devices as d WHERE c.idevice IN(d.id);`)
	for rows.Next() {
		_ = rows.Scan(&temp.Region, &temp.Area, &temp.Idevice, &dStr)
		_ = json.Unmarshal([]byte(dStr), &temp.Device)
		devices = append(devices, temp)
	}
	return devices
}