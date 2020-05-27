package handlers

import (
	"github.com/JanFant/newTLServer/internal/model/crossEdit"
	"github.com/JanFant/newTLServer/internal/model/data"
	u "github.com/JanFant/newTLServer/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

//DisplayDeviceLogFile обработчик отображения файлов лога устройства
var DisplayDeviceLogFile = func(c *gin.Context) {
	mapContx := u.ParserInterface(c.Value("info"))
	resp := crossEdit.DisplayDeviceLog(mapContx, data.GetDB())
	data.CacheInfo.Mux.Lock()
	resp.Obj["regionInfo"] = data.CacheInfo.MapRegion
	resp.Obj["areaInfo"] = data.CacheInfo.MapArea
	data.CacheInfo.Mux.Unlock()
	u.SendRespond(c, resp)
}

//LogDeviceInfo обработчик запроса на выгрузку информации логов устройства за определенный период
var LogDeviceInfo = func(c *gin.Context) {
	arm := &crossEdit.DeviceLogInfo{}
	if err := c.ShouldBindJSON(&arm); err != nil {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "Invalid request"))
		return
	}

	mapContx := u.ParserInterface(c.Value("info"))
	resp := crossEdit.DisplayDeviceLogInfo(*arm, mapContx, data.GetDB())
	u.SendRespond(c, resp)
}