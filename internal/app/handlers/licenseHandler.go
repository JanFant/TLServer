package handlers

import (
	"github.com/JanFant/newTLServer/internal/model/license"
	u "github.com/JanFant/newTLServer/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

//LicenseInfo обработчик сборки начальной информации
var LicenseInfo = func(c *gin.Context) {
	resp := license.LicenseInfo()
	u.SendRespond(c, resp)
}

//LicenseNewKey обработчик обработчик сохранения нового токена
var LicenseNewKey = func(c *gin.Context) {
	type keyStr struct {
		Key string `json:"keyStr"`
	}
	var key keyStr
	if err := c.ShouldBindJSON(&key); err != nil {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "Invalid request"))
		return
	}
	resp := license.LicenseNewKey(key.Key)
	u.SendRespond(c, resp)
}