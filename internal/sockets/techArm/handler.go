package techArm

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ruraomsk/TLServer/internal/model/accToken"
	u "github.com/ruraomsk/TLServer/internal/utils"
	"net/http"
	"strconv"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//HTechArm обработчик открытия сокета
func HTechArm(c *gin.Context, hub *HubTechArm) {
	region := c.Query("Region")
	var reg int
	if region != "" {
		var err error
		reg, err = strconv.Atoi(region)
		if err != nil {
			u.SendRespond(c, u.Message(http.StatusBadRequest, "blank field: Region"))
			return
		}
	} else {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "blank field: Region"))
		return
	}

	area := c.QueryArray("Area")
	if len(area) <= 0 {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "blank field: Area"))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "bad socket connect"))
		return
	}
	accTK, _ := c.Get("tk")
	accInfo, _ := accTK.(*accToken.Token)

	var armInfo = ArmInfo{
		Region:  reg,
		Area:    area,
		AccInfo: accInfo,
	}

	client := &ClientTechArm{hub: hub, conn: conn, send: make(chan armResponse, 256), armInfo: &armInfo}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
