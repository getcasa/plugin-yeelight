package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	portscanner "github.com/anvie/port-scanner"
	"github.com/getcasa/sdk"
)

func main() {
	OnStart()
	for true {
	}
}

const (
	discoverMSG = "M-SEARCH * HTTP/1.1\r\n HOST:239.255.255.250:1982\r\n MAN:\"ssdp:discover\"\r\n ST:wifi_bulb\r\n"

	// timeout value for TCP and UDP commands
	timeout = time.Second * 3

	//SSDP discover address
	ssdpAddr     = "239.255.255.250:1982"
	yeelightPort = 55443

	//CR-LF delimiter
	crlf = "\r\n"
)

// Config define parameters for plugin
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
			Fields: []string{"address", "status"},
		},
		sdk.Action{
			Name:   "toggle",
			Fields: []string{"address"},
		},
	},
}

var (
	discover bool
	lights   []*Yeelight
	Socket   *net.UDPConn
)

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
		ID        string
		Model     string
		Power     string
		Bright    int
		ColorMode int
		CT        int
		RGB       int
		Hue       int
		Sat       int
		Name      string
		Addr      string
		rnd       *rand.Rand
		Socket    net.Conn
		Connected bool
	}
)

// Params define actions parameters available
type Params struct {
	Address string
	Status  bool
}

// OnStart start UDP server to get Xiaomi data
func OnStart() {
	Socket = nil
	discover = true
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

	yee := findLightWithAddr(req.Address)
	if yee == nil {
		return
	}
	fmt.Println("XXXXXXXX")
	fmt.Println(yee.Addr)
	fmt.Println("XXXXXXXX")

	// use name to call actions
	switch name {
	case "setpower":
		yee.SetPower(req.Status)
	case "stripe":
		if yee.Power == "on" {
			yee.SetPower(false)
			yee.Power = "off"
		} else {
			yee.SetPower(true)
			yee.Power = "on"
		}
	default:
		return
	}
}

// OnStop close connection
func OnStop() {
	lights = nil
	discover = false
}

func containIPAddress(arr []string, search string) bool {
	for _, addr := range arr {
		if addr == search {
			return true
		}
	}
	return false
}

var wg sync.WaitGroup

//Discover discovers device in local network via ssdp
func Discover() {

	var ipAddresses []string
	ifaces, err := net.Interfaces()
	fmt.Println(err)

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				cleanAddr := addr.String()[:strings.Index(addr.String(), "/")]
				if cleanAddr != "127.0.0.1" && !strings.Contains(cleanAddr, ":") && !net.ParseIP(cleanAddr).IsLoopback() {
					cleanAddr = addr.String()[:strings.LastIndex(addr.String(), ".")+1]
					if !containIPAddress(ipAddresses, cleanAddr) {
						ipAddresses = append(ipAddresses, cleanAddr)
					}
				}
			}
		}
	}

	wg.Add(len(ipAddresses) * 255)

	for _, ipAddr := range ipAddresses {
		for i := 0; i < 255; i++ {
			go func(i int, ipAddr string) {
				ip := ipAddr + strconv.Itoa(i)
				ps := portscanner.NewPortScanner(ip, 500000*time.Microsecond, 4)
				opened := ps.IsOpen(55443)
				if opened {
					fmt.Println(ip)
					if findLightWithAddr(ip+":"+strconv.Itoa(yeelightPort)) == nil {
						newyee := New(ip + ":" + strconv.Itoa(yeelightPort))
						if newyee != nil {
							newyee.connect()
							// lights = append(lights, newyee)
						}
					}
				}
				wg.Done()
			}(i, ipAddr)

		}
	}
	wg.Wait()
	return
}

func (y *Yeelight) connect() {
	var err error
	notifCh := make(chan *Notification)
	done := make(chan struct{}, 1)
	if y.Socket != nil {
		y.disconnect()
	}
	y.Socket, err = net.DialTimeout("tcp", y.Addr, timeout)
	if err != nil {
		fmt.Println("Error connection: " + y.Addr)
		return
	}

	fmt.Println("Connected: " + y.Addr)
	y.Connected = true

	on, err := y.GetProp("power", "color_mode", "ct", "rgb", "hue", "sat", "bright")
	fmt.Println(err)
	if err == nil {
		fmt.Printf("Power is %s", on)
		// y.SetPower(false)
	}

	go func(c net.Conn) {
		//make sure connection is closed when method returns
		defer y.disconnect()

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

	}(y.Socket)

	// fmt.Println(notifCh.Params)
	// fmt.Println(done)
}

func (y *Yeelight) disconnect() {
	y.Socket.Close()
	y.Socket = nil
	y.Connected = false
}

//New creates new device instance for address provided
func New(addr string) *Yeelight {
	return &Yeelight{
		Addr: addr,
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}

}

// Listen connects to device and listens for NOTIFICATION events
func (y *Yeelight) Listen() (<-chan *Notification, chan<- struct{}, error) {
	var err error
	notifCh := make(chan *Notification)
	done := make(chan struct{}, 1)

	conn, err := net.Dial("tcp", y.Addr)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to %s. %s", y.Addr, err)
	}

	fmt.Println("Connection established " + y.Addr)
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

func findLightWithAddr(addr string) *Yeelight {
	if len(lights) == 0 {
		return nil
	}
	for _, light := range lights {
		if light.Addr == addr || light.ID == addr {
			return light
		}
	}
	return nil
}

// //New creates new device instance for address provided
// func New(addr string, info string) *Yeelight {
// 	if strings.HasSuffix(info, crlf) {
// 		info = info + crlf
// 	}
// 	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(info)), nil)
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	bright, err := strconv.Atoi(resp.Header.Get("BRIGHT"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	ct, err := strconv.Atoi(resp.Header.Get("CT"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	colormode, err := strconv.Atoi(resp.Header.Get("COLOR_MODE"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	rgb, err := strconv.Atoi(resp.Header.Get("RGB"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	hue, err := strconv.Atoi(resp.Header.Get("HUE"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	sat, err := strconv.Atoi(resp.Header.Get("SAT"))
// 	if err != nil {
// 		fmt.Println(err)
// 		return nil
// 	}
// 	return &Yeelight{
// 		ID:        resp.Header.Get("ID"),
// 		Model:     resp.Header.Get("MODEL"),
// 		Power:     resp.Header.Get("POWER"),
// 		Bright:    bright,
// 		ColorMode: colormode,
// 		CT:        ct,
// 		RGB:       rgb,
// 		Hue:       hue,
// 		Sat:       sat,
// 		Name:      resp.Header.Get("NAME"),
// 		Addr:      addr,
// 		rnd:       rand.New(rand.NewSource(time.Now().UnixNano())),
// 	}

// }

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

//parseAddr parses address from ssdp response
func parseAddr(msg string) string {
	if strings.HasSuffix(msg, crlf) {
		msg = msg + crlf
	}
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(msg)), nil)
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

	if !y.Connected || y.Socket == nil {
		y.connect()
	}
	//write request/command
	b, _ := json.Marshal(cmd)
	// fmt.Fprint(y.Socket, string(b)+crlf)

	b = bytes.Join([][]byte{b, []byte(crlf)}, nil)

	y.Socket.Write(b)

	return nil, nil

	// 	//wait and read for response
	// 	res, err := bufio.NewReader(y.Socket).ReadString('\n')
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot read command result %s", err)
	// 	}
	// 	var rs CommandResult
	// 	err = json.Unmarshal([]byte(res), &rs)
	// 	if nil != err {
	// 		return nil, fmt.Errorf("cannot parse command result %s", err)
	// 	}
	// 	if nil != rs.Error {
	// 		return nil, fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message)
	// 	}
	// 	return &rs, nil
}

func (y *Yeelight) randID() int {
	i := y.rnd.Intn(100)
	return i
}

//closeConnection closes network connection
func closeConnection(c net.Conn) {
	if nil != c {
		c.Close()
	}
}
