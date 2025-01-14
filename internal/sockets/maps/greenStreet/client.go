package greenStreet

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/ruraomsk/TLServer/internal/app/tcpConnect"
	"github.com/ruraomsk/TLServer/internal/model/accToken"
	"github.com/ruraomsk/TLServer/internal/model/data"
	"github.com/ruraomsk/TLServer/internal/model/device"
	"github.com/ruraomsk/TLServer/internal/model/routeGS"
	"github.com/ruraomsk/TLServer/internal/sockets"
	"github.com/ruraomsk/TLServer/internal/sockets/maps"
	"github.com/ruraomsk/TLServer/logger"
	"github.com/ruraomsk/ag-server/comm"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	crossPeriod         = time.Second * 2
	devicePeriod        = time.Second * 2
	checkTokensValidity = time.Minute * 1
)

//ClientGS информация о подключившемся пользователе
type ClientGS struct {
	hub        *HubGStreet
	conn       *websocket.Conn
	send       chan gSResponse
	cInfo      *accToken.Token
	devices    []int
	sendPhases bool
}
type Phase struct {
	Device int `json:"device"`
	Phase  int `json:"phase"`
}

//readPump обработчик чтения сокета
func (c *ClientGS) readPump() {
	//db := data.GetDB("ClientGS")
	//defer data.FreeDB("ClientGS")
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	{
		//logger.Debug.Printf("Клиент %s",c.cInfo.Login)
		resp := newGSMess(typeMapInfo, maps.MapOpenInfo())
		//logger.Debug.Printf("Клиент %s mapOpenInfo",c.cInfo.Login)
		resp.Data["routes"] = getAllModes()
		//logger.Debug.Printf("Клиент %s getAllModes",c.cInfo.Login)
		data.CacheArea.Mux.Lock()
		resp.Data["areaZone"] = data.CacheArea.Areas
		data.CacheArea.Mux.Unlock()
		//logger.Debug.Printf("Клиент %s Areas",c.cInfo.Login)
		if c.sendPhases {
			resp.Data[typePhases] = getPhases(c.devices)
			//logger.Debug.Printf("Клиент %s Phases",c.cInfo.Login)
		}
		c.send <- resp
		//logger.Debug.Print("Send %s",c.cInfo.Login)
	}

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			c.hub.unregister <- c
			break
		}
		//ну отправка и отправка
		typeSelect, err := sockets.ChoseTypeMessage(p)
		if err != nil {
			logger.Error.Printf("|IP: %v |Login: %v |Resource: /greenStreet |Message: %v \n", c.cInfo.IP, c.cInfo.Login, err.Error())
			resp := newGSMess(typeError, nil)
			resp.Data["message"] = ErrorMessage{Error: errParseType}
			c.send <- resp
			continue
		}
		switch typeSelect {
		case typeCreateRout:
			{
				temp := routeGS.Route{}
				_ = json.Unmarshal(p, &temp)
				resp := newGSMess(typeCreateRout, nil)
				err := temp.Create()

				if err != nil {
					resp.Data[typeError] = err.Error()
					c.send <- resp
				} else {
					resp.Data["route"] = temp
					resp.Data["login"] = c.cInfo.Login
					c.hub.broadcast <- resp
				}
			}
		case typeUpdateRout:
			{
				temp := routeGS.Route{}
				_ = json.Unmarshal(p, &temp)
				resp := newGSMess(typeUpdateRout, nil)
				err := temp.Update()
				if err != nil {
					resp.Data[typeError] = err.Error()
					c.send <- resp
				} else {
					resp.Data["route"] = temp
					c.hub.broadcast <- resp
				}
			}
		case typeDeleteRout:
			{
				temp := routeGS.Route{}
				_ = json.Unmarshal(p, &temp)
				resp := newGSMess(typeDeleteRout, nil)
				err := temp.Delete()
				if err != nil {
					resp.Data[typeError] = err.Error()
					c.send <- resp
				} else {
					resp.Data["route"] = temp
					c.hub.broadcast <- resp
				}
			}
		case typeJump: //отправка default
			{
				location := &data.Locations{}
				_ = json.Unmarshal(p, &location)
				box, _ := location.MakeBoxPoint()
				resp := newGSMess(typeJump, nil)
				resp.Data["boxPoint"] = box
				c.send <- resp
			}
		case typeDButton: //отправка сообщения о изменениии режима работы
			{
				arm := comm.CommandARM{}
				_ = json.Unmarshal(p, &arm)
				arm.User = c.cInfo.Login
				var mess = tcpConnect.TCPMessage{
					User:        arm.User,
					TCPType:     tcpConnect.TypeDispatch,
					Idevice:     arm.ID,
					Data:        arm,
					From:        tcpConnect.FromCrossSoc,
					CommandType: typeDButton,
					Pos:         sockets.PosInfo{},
				}
				mess.SendToTCPServer()
			}
		case typeRoute:
			{
				execRoute := executeRoute{}
				_ = json.Unmarshal(p, &execRoute)

				arm := comm.CommandARM{Command: 4, User: c.cInfo.Login}
				var mess = tcpConnect.TCPMessage{
					User:        c.cInfo.Login,
					TCPType:     tcpConnect.TypeDispatch,
					From:        tcpConnect.FromCrossSoc,
					CommandType: typeDButton,
					Pos:         sockets.PosInfo{},
				}
				if execRoute.TurnOn {
					c.sendPhases = true
					c.devices = execRoute.Devices
					//logger.Debug.Printf("client devs %v",c.devices)
					arm.Params = 1
					device.GlobalDevEdit.Mux.Lock()
					for _, dev := range execRoute.Devices {
						tDev := device.GlobalDevEdit.MapDevices[dev]
						if tDev.BusyCount == 0 || tDev.TurnOnFlag == false {
							arm.ID = dev
							mess.Idevice = arm.ID
							mess.Data = arm
							mess.SendToTCPServer()
							tDev.TurnOnFlag = true
						}
						tDev.BusyCount++
						device.GlobalDevEdit.MapDevices[dev] = tDev
					}
					device.GlobalDevEdit.Mux.Unlock()
				} else {
					c.sendPhases = false
					arm.Params = 0
					device.GlobalDevEdit.Mux.Lock()
					for _, dev := range c.devices {
						tDev := device.GlobalDevEdit.MapDevices[dev]
						tDev.BusyCount--
						if tDev.BusyCount == 0 && tDev.TurnOnFlag == true {
							arm.ID = dev
							mess.Idevice = arm.ID
							mess.Data = arm
							mess.SendToTCPServer()
							tDev.TurnOnFlag = false
						}
						device.GlobalDevEdit.MapDevices[dev] = tDev
					}
					device.GlobalDevEdit.Mux.Unlock()
					c.devices = make([]int, 0)
				}
			}
		default:
			{
				resp := newGSMess("type", nil)
				resp.Data["type"] = typeSelect
				c.send <- resp
			}
		}
	}
}

//writePump обработчик записи в сокет
func (c *ClientGS) writePump() {
	pingTick := time.NewTicker(pingPeriod)
	defer func() {
		pingTick.Stop()
	}()
	for {
		select {
		case mess, ok := <-c.send:
			{
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "канал был закрыт"))
					return
				}
				_ = c.conn.WriteJSON(mess)
				// Add queued chat messages to the current websocket message.
				n := len(c.send)
				for i := 0; i < n; i++ {
					_ = c.conn.WriteJSON(<-c.send)
				}
			}
		case <-pingTick.C:
			{
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}
