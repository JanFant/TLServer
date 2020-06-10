package handlers

import (
	"net/http"

	"github.com/JanFant/TLServer/internal/model/crossEdit"
	"github.com/JanFant/TLServer/internal/model/data"
	u "github.com/JanFant/TLServer/internal/utils"
	"github.com/gin-gonic/gin"
)

//PosInfo сбор информации о занятых перекрестках
var CrossEditInfo = func(c *gin.Context) {
	mapContx := u.ParserInterface(c.Value("info"))
	resp := crossEdit.DisplayCrossEditInfo(mapContx, data.GetDB())
	data.CacheInfo.Mux.Lock()
	resp.Obj["regionInfo"] = data.CacheInfo.MapRegion
	resp.Obj["areaInfo"] = data.CacheInfo.MapArea
	data.CacheInfo.Mux.Unlock()
	u.SendRespond(c, resp)
}

//CrossEditFree освобождение перекрестков
var CrossEditFree = func(c *gin.Context) {
	var busyArms crossEdit.BusyArms
	if err := c.ShouldBindJSON(&busyArms); err != nil {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "invalid request"))
		return
	}
	resp := crossEdit.FreeCrossEdit(busyArms)
	u.SendRespond(c, resp)
}
