package tcpConnect

import (
	"net"
	"time"

	"github.com/JanFant/TLServer/logger"
)

//StateChan канал для передачи информации связанной со state
var StateChan = make(chan StateMessage)

//ArmCommandChan канал для передачи информации связанной с командами арма
var ArmCommandChan = make(chan ArmCommandMessage)

//StateMessage state информация для отправки на сервер
type StateMessage struct {
	User     string //пользователь отправляющий данные (логин)
	Info     string //короткая информация о state
	StateStr string //данные подготовленные к отправке
	Message  string //информация о результате передачи данных
}

//ArmCommandMessage ARM информация для отправки на сервер
type ArmCommandMessage struct {
	User       string //пользователь отправляющий данные (логин)
	CommandStr string //данные подготовленные к отправке
	Message    string //информация о результате передачи данных
}

//TCPConfig настройки для тсп соединения
type TCPConfig struct {
	ServerAddr  string `toml:"tcpServerAddress"` //адресс сервера
	PortState   string `toml:"portState"`        //порт для обмена Стате
	PortArmComm string `toml:"portArmCommand"`   //порт для обмена арм командами
}

//getStateIP возвращает ip+port для State соединения
func (tcpConfig *TCPConfig) getStateIP() string {
	return tcpConfig.ServerAddr + tcpConfig.PortState
}

//getArmIP возвращает ip+port для ArmCommand соединения
func (tcpConfig *TCPConfig) getArmIP() string {
	return tcpConfig.ServerAddr + tcpConfig.PortArmComm
}

//TCPClientStart запуск соединений
func TCPClientStart(tcpConfig TCPConfig) {
	typeInfo = make(map[string]string)
	typeInfo[TypeDispatch] = tcpConfig.getArmIP()
	typeInfo[TypeState] = tcpConfig.getStateIP()
	go TCPBroadcast(typeInfo)
}

//TCPForState обмен с сервером данными State
func TCPForState(IP string) {
	var (
		conn     net.Conn
		err      error
		errCount = 0
	)
	timeTick := time.NewTicker(time.Second * 5)
	defer timeTick.Stop()
	FlagConnect := false
	for {
		select {
		case state := <-StateChan:
			{
				if !FlagConnect {
					state.Message = "TCP Server not responding"
					StateChan <- state
					continue
				}
				state.StateStr += "\n"
				_ = conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
				_, err := conn.Write([]byte(state.StateStr))
				if err != nil {
					if errCount < 5 {
						logger.Error.Println("|Message: TCP Server " + IP + " not responding: " + err.Error())
						FlagConnect = false
					}
					errCount++
					state.Message = err.Error()
					StateChan <- state
					_ = conn.Close()
					break
				}
				state.Message = "ok"
				errCount = 0
				StateChan <- state
			}
		case <-timeTick.C:
			{
				if !FlagConnect {
					conn, err = net.Dial("tcp", IP)
					if err != nil {
						if errCount < 5 {
							logger.Error.Println("|Message: TCP Server " + IP + " not responding: " + err.Error())
						}
						errCount++
						time.Sleep(time.Second * 5)
						continue
					}
					FlagConnect = true
				}
				_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
				_, err := conn.Write([]byte("0\n"))
				if err != nil {
					FlagConnect = false
				}
			}
		}
	}
}

//TCPForARM обмен с сервером командами для АРМ
func TCPForARM(IP string) {
	var (
		conn     net.Conn
		err      error
		errCount = 0
	)
	timeTick := time.NewTicker(time.Second * 5)
	defer timeTick.Stop()
	FlagConnect := false
	for {
		select {
		case armCommand := <-ArmCommandChan:
			{
				if !FlagConnect {
					armCommand.Message = "TCP Server not responding"
					ArmCommandChan <- armCommand
					continue
				}
				armCommand.CommandStr += "\n"
				_ = conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
				_, err := conn.Write([]byte(armCommand.CommandStr))
				if err != nil {
					if errCount < 5 {
						logger.Error.Println("|Message: TCP Server " + IP + " not responding: " + err.Error())
						FlagConnect = false
					}
					errCount++
					armCommand.Message = err.Error()
					ArmCommandChan <- armCommand
					_ = conn.Close()
					break
				}
				armCommand.Message = "ok"
				errCount = 0
				ArmCommandChan <- armCommand
			}
		case <-timeTick.C:
			{
				if !FlagConnect {
					conn, err = net.Dial("tcp", IP)
					if err != nil {
						if errCount < 5 {
							logger.Error.Println("|Message: TCP Server " + IP + " not responding: " + err.Error())
						}
						errCount++
						time.Sleep(time.Second * 5)
						continue
					}
					FlagConnect = true
				}
				_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
				_, err := conn.Write([]byte("0\n"))
				if err != nil {
					FlagConnect = false
				}
			}
		}
	}
}
