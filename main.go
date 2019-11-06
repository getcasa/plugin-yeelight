package main

import (
	"bufio"
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
	OnStart(nil)
	for range time.Tick(5 * time.Second) {
		for _, li := range lights {
			fmt.Println(li.ID)
			li.Toggle()
			// fmt.Println(li.Name)
			// fmt.Println(li.Model)
			// fmt.Println(li.Power)
			// fmt.Println("--")
		}
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
			Name: "setpower",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "address",
					Type:   "string",
					Config: true,
				},
				sdk.Field{
					Name:   "state",
					Type:   "string",
					Config: true,
				},
			},
		},
		sdk.Action{
			Name: "toggle",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "address",
					Type:   "string",
					Config: true,
				},
			},
		},
	},
}

var (
	lights []*Yeelight
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
		ID         string
		Model      string
		Power      string
		Bright     int
		ColorMode  int
		CT         int
		RGB        int
		Hue        int
		Sat        int
		Flowing    int
		DelayOff   int
		FlowParams int
		MusicOn    int
		Name       string
		Addr       string
		rnd        *rand.Rand
		Socket     net.Conn
		Connected  bool
	}
)

// Params define actions parameters available
type Params struct {
	Address string
	State   bool
}

// Init
func Init() []byte {
	return []byte{}
}

// OnStart start UDP server to get Xiaomi data
func OnStart(config []byte) {
	go Discover()

	return
}

// OnData get data from xiaomi gateway
func OnData() interface{} {
	return nil
}

// CallAction call functions from actions
func CallAction(name string, params []byte, config []byte) {
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
		yee.SetPower(req.State)
	case "toggle":
		yee.Toggle()
	default:
		return
	}
}

// OnStop close connection
func OnStop() {
	for _, light := range lights {
		light.disconnect()
	}
	lights = nil
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

//Discover discovers device in local network
func Discover() []sdk.Device {
	var devices []sdk.Device

	for range time.Tick(timeout) {
		discover()
	}
	// for _, light := range lights {
	// 	device := sdk.Device{

	// 	}
	// }

	return devices
}

//discover discovers device in local network via ssdp
func discover() {
	var ipAddresses []string
	ifaces, err := net.Interfaces()
	if err == nil {

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
					if findLightWithAddr(ip+":"+strconv.Itoa(yeelightPort)) == nil {
						ps := portscanner.NewPortScanner(ip, 4*time.Second, 4)
						opened := ps.IsOpen(55443)
						if opened {
							newyee := New(ip + ":" + strconv.Itoa(yeelightPort))
							if newyee != nil {
								newyee.connect()
								if newyee.Connected {
									lights = append(lights, newyee)
								}
							}
						}
					}
					wg.Done()
				}(i, ipAddr)

			}
		}
		wg.Wait()
	}

	check := false
	for _, li := range lights {
		if li.ID == "" {
			check = true
			break
		}
	}

	if check {
		for check {
			ssdp, _ := net.ResolveUDPAddr("udp4", ssdpAddr)
			c, _ := net.ListenPacket("udp4", ":0")
			socket := c.(*net.UDPConn)
			socket.WriteToUDP([]byte(discoverMSG), ssdp)
			socket.SetReadDeadline(time.Now().Add(timeout))

			rsBuf := make([]byte, 1024)
			size, _, err := socket.ReadFromUDP(rsBuf)
			if err != nil {
				fmt.Println(err)
			}
			rs := rsBuf[0:size]
			addr := parseAddr(string(rs))
			id := parseID(string(rs))
			light := findLightWithAddr(addr)
			if light != nil && light.ID == "" {
				findLightWithAddr(addr).ID = id
			}
		}
	}
}

func (y *Yeelight) stayActive() {
	for range time.Tick(20 * time.Second) {
		y.Update()
	}
}

func (y *Yeelight) connect() {
	var err error
	// notifCh := make(chan *Notification)
	// done := make(chan struct{}, 1)
	if y.Socket != nil {
		y.disconnect()
	}
	y.Socket, err = net.DialTimeout("tcp", y.Addr, 5*time.Second)
	if err != nil {
		fmt.Println("Error connection: " + y.Addr)
		return
	}

	fmt.Println("Connected: " + y.Addr)
	y.Connected = true

	status := y.Update()
	if !status {
		y.disconnect()
		return
	}

	info, err := y.GetProp("model", "name", "id")
	if err == nil {
		if len(info) >= 1 {
			y.Model = info[0].(string)
		}
		if len(info) >= 2 {
			y.Name = info[1].(string)
		}
	}

	go y.stayActive()

	// go func(c net.Conn) {
	// 	//make sure connection is closed when method returns
	// 	defer y.disconnect()

	// 	connReader := bufio.NewReader(c)
	// 	for {
	// 		select {
	// 		case <-done:
	// 			return
	// 		default:
	// 			data, err := connReader.ReadString('\n')
	// 			if nil == err {
	// 				var rs Notification
	// 				fmt.Println(data)
	// 				json.Unmarshal([]byte(data), &rs)
	// 				select {
	// 				case notifCh <- &rs:
	// 				default:
	// 					fmt.Println("Channel is full")
	// 				}
	// 			}
	// 		}

	// 	}

	// }(y.Socket)
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

//Update update yeelight info
func (y *Yeelight) Update() bool {

	if !y.Connected || y.Socket == nil {
		y.connect()
		if y.Socket == nil {
			return false
		}
	}

	on, err := y.GetProp("power", "color_mode", "ct", "rgb", "hue", "sat", "bright", "flowing", "delayoff", "flow_params", "music_on")
	if err == nil {
		if len(on) >= 1 {
			y.Power = on[0].(string)
		}
		if len(on) >= 2 {
			y.ColorMode, _ = strconv.Atoi(on[1].(string))
		}
		if len(on) >= 3 {
			y.CT, _ = strconv.Atoi(on[2].(string))
		}
		if len(on) >= 4 {
			y.RGB, _ = strconv.Atoi(on[3].(string))
		}
		if len(on) >= 5 {
			y.Hue, _ = strconv.Atoi(on[4].(string))
		}
		if len(on) >= 6 {
			y.Sat, _ = strconv.Atoi(on[5].(string))
		}
		if len(on) >= 7 {
			y.Bright, _ = strconv.Atoi(on[6].(string))
		}
		if len(on) >= 8 {
			y.Flowing, _ = strconv.Atoi(on[7].(string))
		}
		if len(on) >= 9 {
			y.DelayOff, _ = strconv.Atoi(on[8].(string))
		}
		if len(on) >= 10 {
			y.FlowParams, _ = strconv.Atoi(on[9].(string))
		}
		if len(on) >= 11 {
			y.MusicOn, _ = strconv.Atoi(on[10].(string))
		}
		return true
	}
	return false
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

//Toggle is used to switch on or off the smart LED (software managed on/off).
func (y *Yeelight) Toggle() error {
	_, err := y.executeCommand("toggle")
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

//parseID parses address from ssdp response
func parseID(msg string) string {
	if strings.HasSuffix(msg, crlf) {
		msg = msg + crlf
	}
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(msg)), nil)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get("ID")
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
		if y.Socket == nil {
			return nil, nil
		}
	}

	y.Socket.SetReadDeadline(time.Now().Add(timeout))

	//write request/command
	b, _ := json.Marshal(cmd)
	fmt.Fprint(y.Socket, string(b)+crlf)

	//wait and read for response
	// res, err := bufio.NewReader(y.Socket).ReadString('\n')
	reply := make([]byte, 1024)
	size, err := y.Socket.Read(reply)
	if err != nil {
		return nil, fmt.Errorf("cannot read command result %s", err)
	}

	reply = reply[:size]

	var rs CommandResult
	err = json.Unmarshal(reply, &rs)
	if nil != err {
		return nil, fmt.Errorf("cannot parse command result %s", err)
	}
	if nil != rs.Error {
		return nil, fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message)
	}

	fmt.Println(rs.Result)
	return &rs, nil

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
