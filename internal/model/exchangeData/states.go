package exchangeData

import (
	"github.com/JanFant/TLServer/internal/model/license"
	"github.com/JanFant/TLServer/internal/sockets/crossSock"
	u "github.com/JanFant/TLServer/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/ruraomsk/ag-server/pudge"
	"net/http"
)

func GetStates(iDevice []int, db *sqlx.DB) u.Response {
	var (
		statesList = make([]pudge.Cross, 0)
	)

	query, args, err := sqlx.In(`SELECT state FROM public.cross WHERE idevice IN (?)`, iDevice)
	if err != nil {
		return u.Message(http.StatusInternalServerError, "error formatting IN query")
	}
	query = db.Rebind(query)
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return u.Message(http.StatusInternalServerError, "db server not response")
	}

	for rows.Next() {
		var stateStr string
		err = rows.Scan(&stateStr)
		if err != nil {
			return u.Message(http.StatusInternalServerError, "error convert cross info")
		}

		state, _ := crossSock.ConvertStateStrToStruct(stateStr)

		statesList = append(statesList, state)
	}

	//обережим количество устройств по количеству доступному в лицензии
	numDev := license.LicenseFields.NumDev
	if len(statesList) > numDev {
		statesList = statesList[:numDev]
	}

	resp := u.Response{Code: http.StatusOK, Obj: map[string]interface{}{}}
	resp.Obj["data"] = statesList
	return resp
}