package xctrl

var (
	typeXctrlChange = "xctrlChange"
	typeXctrlCreate = "xctrlCreate"
	typeXctrlInfo   = "xctrlInfo"
	typeXctrlReInfo = "xctrlReInfo"
	typeXctrlUpdate = "xctrlUpdate"
	typeXctrlDelete = "xctrlDelete"

	typeError   = "error"
	typeClose   = "close"
	typeGetArea = "getArea"

	errParseType = "Сервер не смог обработать запрос"
	errBD        = "Ошибка обращения к БД"
)

//MessXctrl структура пакета сообщения для xctrl
type MessXctrl struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

//newXctrlMess создание
func newXctrlMess(mType string, data map[string]interface{}) MessXctrl {
	var resp MessXctrl
	resp.Type = mType
	if data != nil {
		resp.Data = data
	} else {
		resp.Data = make(map[string]interface{})
	}
	return resp
}

//ErrorMessage структура ошибки
type ErrorMessage struct {
	Error string `json:"error"`
}
