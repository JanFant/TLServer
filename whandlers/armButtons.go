package whandlers

import (
	"../data"
	u "../utils"
	"encoding/json"
	"github.com/ruraomsk/ag-server/comm"
	"net/http"
)

//DispatchControlButton обработчик кнопок диспетчерского управления
var DispatchControlButtons = func(w http.ResponseWriter, r *http.Request) {
	flag, resp := FuncAccessCheck(w, r, "ControlCross")
	if flag {
		arm := comm.CommandARM{}
		if err := json.NewDecoder(r.Body).Decode(&arm); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			u.Respond(w, r, u.Message(false, "Invalid request"))
			return
		}
		mapContx := u.ParserInterface(r.Context().Value("info"))
		resp = data.DispatchControl(arm, mapContx)
	}
	u.Respond(w, r, resp)
}