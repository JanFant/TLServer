package chat

import (
	"time"
)

//HubChat структура хаба для cross
type HubChat struct {
	clients    map[*ClientChat]bool
	broadcast  chan chatResponse
	register   chan *ClientChat
	unregister chan *ClientChat
}

//NewChatHub создание хаба
func NewChatHub() *HubChat {
	return &HubChat{
		broadcast:  make(chan chatResponse),
		clients:    make(map[*ClientChat]bool),
		register:   make(chan *ClientChat),
		unregister: make(chan *ClientChat),
	}
}

//Run запуск хаба для Chat
func (h *HubChat) Run() {

	UserLogoutChat = make(chan string)
	checkValidityTicker := time.NewTicker(checkTokensValidity)
	defer checkValidityTicker.Stop()
	for {
		select {
		case client := <-h.register:
			{
				//проверяем нужно ли оповещать других пользоветелей о подключенном
				flagNew := true
				for hClient := range h.clients {
					if hClient.clientInfo.accInfo.Login == client.clientInfo.accInfo.Login {
						flagNew = false
						break
					}
				}
				if flagNew {
					resp := newChatMess(typeStatus, nil)
					resp.Data["user"] = client.clientInfo.accInfo.Login
					resp.Data["status"] = client.clientInfo.status
					for hClient := range h.clients {
						if hClient.clientInfo.accInfo.Login != client.clientInfo.accInfo.Login {
							hClient.send <- resp
						}
					}
				}

				h.clients[client] = true
				//отправим собранные данные клиенту
			}
		case client := <-h.unregister:
			{
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
					_ = client.conn.Close()

					flagOffline := true
					for hClient := range h.clients {
						if hClient.clientInfo.accInfo.Login == client.clientInfo.accInfo.Login {
							flagOffline = false
							break
						}
					}
					if flagOffline {
						resp := newChatMess(typeStatus, nil)
						resp.Data["user"] = client.clientInfo.accInfo.Login
						resp.Data["status"] = statusOffline
						for hClient := range h.clients {
							hClient.send <- resp
						}
					}
				}
			}
		case mess := <-h.broadcast:
			{
				if mess.to == globalMessage {
					for client := range h.clients {
						select {
						case client.send <- mess:
						default:
							delete(h.clients, client)
							close(client.send)
						}
					}
				}
				if mess.to != globalMessage {
					for client := range h.clients {
						if mess.to == client.clientInfo.accInfo.Login || mess.from == client.clientInfo.accInfo.Login {
							select {
							case client.send <- mess:
							default:
								delete(h.clients, client)
								close(client.send)
							}
						}
					}
				}
			}
		case login := <-UserLogoutChat:
			{
				for client := range h.clients {
					if client.clientInfo.accInfo.Login == login {
						msg := newChatMess(typeClose, nil)
						msg.Data["message"] = "пользователь вышел из системы"
						client.send <- msg
					}
				}
			}
		case <-checkValidityTicker.C:
			{
				for client := range h.clients {
					if client.clientInfo.accInfo.Valid() != nil {
						msg := newChatMess(typeClose, nil)
						msg.Data["message"] = "вышло время сеанса пользователя"
						client.send <- msg
					}
				}
			}
		}
	}
}
