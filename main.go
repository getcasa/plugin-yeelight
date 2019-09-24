package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/getcasa/sdk"
)

func main() {
	OnStart()

	for len(lights) == 0 {
	}
}

const (
	discoverMSG = "M-SEARCH * HTTP/1.1\r\n HOST:239.255.255.250:1982\r\n MAN:\"ssdp:discover\"\r\n ST:wifi_bulb\r\n"

	// timeout value for TCP and UDP commands
	timeout = time.Second * 10

	//SSDP discover address
	ssdpAddr = "239.255.255.250:1982"

	//CR-LF delimiter
	crlf = "\r\n"
)

var Config = sdk.Configuration{
	Name:        "yeelight",
	Version:     "1.0.0",
	Author:      "amoinier",
	Description: "yeelight",
	Main:        "yeelight",
	FuncData:    "onData",
	Actions: []sdk.Action{
		sdk.Action{
			Name:   "setpower",
			Fields: []string{"Light", "status"},
		},
	},
}

var lights []*Yeelight

type (
	//Command represents COMMAND request to Yeelight device
	Command struct {
		ID     int           `json:"id"`
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
	}

	// CommandResult represents response from Yeelight device
	CommandResult struct {
		ID     int           `json:"id"`
		Result []interface{} `json:"result,omitempty"`
		Error  *Error        `json:"error,omitempty"`
	}

	// Notification represents notification response
	Notification struct {
		Method string            `json:"method"`
		Params map[string]string `json:"params"`
	}

	//Error struct represents error part of response
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	//Yeelight represents device
	Yeelight struct {
		addr string
		rnd  *rand.Rand
	}
)

// Params define actions parameters available
type Params struct {
	Light  *Yeelight
	Status bool
}

// OnStart start UDP server to get Xiaomi data
func OnStart() {
	go Discover()

	return
}

// OnData get data from xiaomi gateway
func OnData() interface{} {
	return nil
}

// CallAction call functions from actions
func CallAction(name string, params []byte) {
	if string(params) == "" {
		fmt.Println("Params must be provided")
		return
	}

	// declare parameters
	var req Params

	// unmarshal parameters to use in actions
	err := json.Unmarshal(params, &req)
	if err != nil {
		fmt.Println(err)
	}

	// use name to call actions
	switch name {
	case "setpower":
		req.Light.SetPower(req.Status)
	case "post":
		// Post(req.Link, req.CtnType, req.Values)
	default:
		return
	}
}

// OnStop close connection
func OnStop() {
}

//Discover discovers device in local network via ssdp
func Discover() {
	var addr string

	for true {
		ssdp, _ := net.ResolveUDPAddr("udp4", ssdpAddr)
		c, _ := net.ListenPacket("udp4", ":0")
		socket := c.(*net.UDPConn)
		socket.WriteToUDP([]byte(discoverMSG), ssdp)
		socket.SetReadDeadline(time.Now().Add(timeout))
		rsBuf := make([]byte, 1024)
		size, _, err := socket.ReadFromUDP(rsBuf)
		if err != nil {
			fmt.Println("no devices found")
		} else {
			rs := rsBuf[0:size]
			fmt.Println(size)
			fmt.Println(string(rs))
			addr = parseAddr(string(rs))
			fmt.Printf("Device with ip %s found\n", addr)
			if !checkAddrInLights(addr) {
				arrayLight := append(lights, New(addr))
				lights = arrayLight
			}
		}
	}

	return
}

func checkAddrInLights(addr string) bool {
	for _, light := range lights {
		if light.addr == addr {
			return true
		}
	}
	return false
}

//New creates new device instance for address provided
func New(addr string) *Yeelight {
	return &Yeelight{
		addr: addr,
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}

}

//parseAddr parses address from ssdp response
func parseAddr(msg string) string {
	if strings.HasSuffix(msg, crlf) {
		msg = msg + crlf
	}
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(msg)), nil)
	fmt.Println("-----")
	fmt.Println(resp.Header.Get("ID"))
	fmt.Println("-----")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer resp.Body.Close()
	return strings.TrimPrefix(resp.Header.Get("LOCATION"), "yeelight://")
}

// GetProp method is used to retrieve current property of smart LED.
func (y *Yeelight) GetProp(values ...interface{}) ([]interface{}, error) {
	r, err := y.executeCommand("get_prop", values...)
	if nil != err {
		return nil, err
	}
	return r.Result, nil
}

//SetPower is used to switch on or off the smart LED (software managed on/off).
func (y *Yeelight) SetPower(on bool) error {
	var status string
	if on {
		status = "on"
	} else {
		status = "off"
	}
	_, err := y.executeCommand("set_power", status)
	return err
}

func (y *Yeelight) newCommand(name string, params []interface{}) *Command {
	return &Command{
		Method: name,
		ID:     y.randID(),
		Params: params,
	}
}

//executeCommand executes command with provided parameters
func (y *Yeelight) executeCommand(name string, params ...interface{}) (*CommandResult, error) {
	return y.execute(y.newCommand(name, params))
}

//executeCommand executes command
func (y *Yeelight) execute(cmd *Command) (*CommandResult, error) {

	conn, err := net.Dial("tcp", y.addr)
	if nil != err {
		return nil, fmt.Errorf("cannot open connection to %s. %s", y.addr, err)
	}
	time.Sleep(time.Second)
	conn.SetReadDeadline(time.Now().Add(timeout))

	//write request/command
	b, _ := json.Marshal(cmd)
	fmt.Fprint(conn, string(b)+crlf)

	//wait and read for response
	res, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("cannot read command result %s", err)
	}
	var rs CommandResult
	err = json.Unmarshal([]byte(res), &rs)
	if nil != err {
		return nil, fmt.Errorf("cannot parse command result %s", err)
	}
	if nil != rs.Error {
		return nil, fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message)
	}
	return &rs, nil
}

func (y *Yeelight) randID() int {
	i := y.rnd.Intn(100)
	return i
}

// Listen connects to device and listens for NOTIFICATION events
func (y *Yeelight) Listen() (<-chan *Notification, chan<- struct{}, error) {
	var err error
	notifCh := make(chan *Notification)
	done := make(chan struct{}, 1)

	conn, err := net.DialTimeout("tcp", y.addr, time.Second*3)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to %s. %s", y.addr, err)
	}

	fmt.Println("Connection established")
	go func(c net.Conn) {
		//make sure connection is closed when method returns
		defer closeConnection(conn)

		connReader := bufio.NewReader(c)
		for {
			select {
			case <-done:
				return
			default:
				data, err := connReader.ReadString('\n')
				if nil == err {
					var rs Notification
					fmt.Println(data)
					json.Unmarshal([]byte(data), &rs)
					select {
					case notifCh <- &rs:
					default:
						fmt.Println("Channel is full")
					}
				}
			}

		}

	}(conn)

	return notifCh, done, nil
}

//closeConnection closes network connection
func closeConnection(c net.Conn) {
	if nil != c {
		c.Close()
	}
}
