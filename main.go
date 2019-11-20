package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	portscanner "github.com/anvie/port-scanner"
	"github.com/getcasa/sdk"
)

func main() {
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
	Description: "Controls yeelight ecosystem",
	Devices: []sdk.Device{
		sdk.Device{
			Name:           "stripe",
			Description:    "",
			DefaultTrigger: "power",
			DefaultAction:  "toggle",
			Triggers: []sdk.Trigger{
				sdk.Trigger{
					Name:          "Power",
					Direct:        false,
					Type:          "string",
					Possibilities: []string{"on", "off"},
				},
				sdk.Trigger{
					Name:   "Bright",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "ColorMode",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "CT",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "RGB",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Hue",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Sat",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Flowing",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "DelayOff",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "FlowParams",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "MusicOn",
					Direct: false,
					Type:   "int",
				},
			},
			Actions: []string{"setpower", "toggle"},
		},
		sdk.Device{
			Name:           "color",
			Description:    "",
			DefaultTrigger: "power",
			DefaultAction:  "toggle",
			Triggers: []sdk.Trigger{
				sdk.Trigger{
					Name:          "Power",
					Direct:        false,
					Type:          "string",
					Possibilities: []string{"on", "off"},
				},
				sdk.Trigger{
					Name:   "Bright",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "ColorMode",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "CT",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "RGB",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Hue",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Sat",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "Flowing",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "DelayOff",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "FlowParams",
					Direct: false,
					Type:   "int",
				},
				sdk.Trigger{
					Name:   "MusicOn",
					Direct: false,
					Type:   "int",
				},
			},
			Actions: []string{"setpower", "toggle"},
		},
	},
	Actions: []sdk.Action{
		sdk.Action{
			Name: "setpower",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "state",
					Type:   "string",
					Config: true,
				},
			},
		},
		sdk.Action{
			Name:   "toggle",
			Fields: []sdk.Field{},
		},
		sdk.Action{
			Name: "set_ct",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "ct_value",
					Type:   "int",
					Config: true,
				},
				sdk.Field{
					Name:   "effect",
					Type:   "string",
					Config: true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Config: true,
				},
			},
		},
	},
}

var (
	lights []*Yeelight
	datas  chan sdk.Data
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

	// Error struct represents error part of response
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
		Stay       bool
	}
)

// Params define actions parameters available
type Params struct {
	State    bool
	CtValue  int    `db:"ct_value" json:"ctValue"`
	Effect   string `db:"effect" json:"effect"`
	Duration int    `db:"duration" json:"duration"`
	Action   int    `db:"action" json:"action"`
	Host     string `db:"host" json:"host"`
	Port     string `db:"port" json:"port"`
}

// OnStart start UDP server to get Xiaomi data
func OnStart(config []byte) {
	datas = make(chan sdk.Data)
	Discover()

	go func() {
		for range time.Tick(5 * time.Second) {
			findIDLight()
		}
	}()

	return
}

// OnData func
func OnData() []sdk.Data {
	toSend := <-datas

	return []sdk.Data{toSend}
}

// CallAction call functions from actions
func CallAction(physicalID string, name string, params []byte, config []byte) {
	if string(params) == "" {
		fmt.Println("Params must be provided")
	}

	// declare parameters
	var req Params

	// unmarshal parameters to use in actions
	json.Unmarshal(params, &req)

	yee := findLightWithAddr(physicalID)
	if yee == nil {
		return
	}

	// use name to call actions
	switch name {
	case "setpower":
		yee.SetPower(req.State)
	case "toggle":
		yee.Toggle()
	case "set_ct":
		yee.StartFunc("set_ct_abx", req.CtValue, req.Effect, req.Duration)
	case "music":
		yee.StartFunc("set_music", req.Action, req.Host, req.Port)
	default:
	}

	go yee.Update()
}

// OnStop close connection
func OnStop() {
	for _, light := range lights {
		light.disconnect()
	}
	lights = nil
}

var wg sync.WaitGroup

//Discover discovers device in local network
func Discover() []sdk.DiscoveredDevice {
	var devices []sdk.DiscoveredDevice

	go discover()
	go findIDLight()

	for _, light := range lights {
		light.Update()
		if light.ID == "" || !light.Connected || light.Socket == nil {
			continue
		}

		devices = append(devices, sdk.DiscoveredDevice{
			Name:         light.Name,
			PhysicalID:   light.ID,
			PhysicalName: light.Model,
			Plugin:       Config.Name,
		})
	}

	return devices
}

//discover discovers device in local network via ssdp
func discover() error {
	var ipAddresses []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			cleanAddr := addr.String()[:strings.Index(addr.String(), "/")]
			if cleanAddr == "127.0.0.1" || strings.Contains(cleanAddr, ":") || net.ParseIP(cleanAddr).IsLoopback() {
				continue
			}
			cleanAddr = addr.String()[:strings.LastIndex(addr.String(), ".")+1]
			if containIPAddress(ipAddresses, cleanAddr) {
				continue
			}
			ipAddresses = append(ipAddresses, cleanAddr)
		}
	}

	wg.Add(len(ipAddresses) * 255)

	for _, ipAddr := range ipAddresses {
		for i := 0; i < 255; i++ {
			go func(i int, ipAddr string) {
				ip := ipAddr + strconv.Itoa(i)
				light := findLightWithAddr(ip + ":" + strconv.Itoa(yeelightPort))
				if light != nil {
					if !light.Connected || light.Socket == nil {
						light.connect()
					}
					wg.Done()
					return
				}
				if !portscanner.NewPortScanner(ip, 10*time.Second, 4).IsOpen(yeelightPort) {
					wg.Done()
					return
				}
				newyee := New(ip + ":" + strconv.Itoa(yeelightPort))
				newyee.connect()
				if newyee == nil || !newyee.Connected || newyee.Socket == nil {
					wg.Done()
					return
				}
				lights = append(lights, newyee)
				wg.Done()
			}(i, ipAddr)
		}
	}
	wg.Wait()

	return nil
}

func containIPAddress(arr []string, search string) bool {
	for _, addr := range arr {
		if addr == search {
			return true
		}
	}
	return false
}

func findIDLight() {
	check := true
	for check {
		time.Sleep(250 * time.Millisecond)
		check = false
		for _, light := range lights {
			if light.ID == "" {
				check = true
				break
			}
		}
		ssdp, _ := net.ResolveUDPAddr("udp4", ssdpAddr)
		c, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			continue
		}
		socket := c.(*net.UDPConn)
		socket.WriteToUDP([]byte(discoverMSG), ssdp)
		socket.SetReadDeadline(time.Now().Add(timeout))

		rsBuf := make([]byte, 1024)
		size, _, err := socket.ReadFromUDP(rsBuf)
		if err != nil {
			continue
		}
		rs := rsBuf[0:size]

		addr := parseAddr(string(rs))
		light := findLightWithAddr(addr)
		if light != nil && light.ID == "" {
			findLightWithAddr(addr).ID = parseID(string(rs))
			fmt.Println(addr + " - " + parseID(string(rs)))
		}
	}
}

func (y *Yeelight) connect() {
	var err error
	if y.Socket != nil {
		y.disconnect()
	}
	y.Socket, err = net.DialTimeout("tcp", y.Addr, 8*time.Second)
	if err != nil {
		fmt.Println("Error connection: " + y.Addr + ", error: " + err.Error())
		return
	}

	fmt.Println("Connected: " + y.Addr)
	y.Connected = true

	status := y.Update()
	if !status {
		y.disconnect()
		return
	}

	info, err := y.GetProp("model", "name")
	if err == nil {
		if len(info) >= 1 {
			y.Model = info[0].(string)
		}
		if len(info) >= 2 {
			y.Name = info[1].(string)
		}
	}

	if !y.Stay {
		y.Stay = true
		go y.stayActive()
	}
}

func (y *Yeelight) stayActive() {
	for range time.Tick(5 * time.Minute) {
		y.Update()
	}
}

func (y *Yeelight) disconnect() {
	if y.Socket != nil {
		y.Socket.Close()
	}
	y.Socket = nil
	y.Connected = false
}

//New creates new device instance for address provided
func New(addr string) *Yeelight {
	return &Yeelight{
		Addr: addr,
		rnd:  rand.New(rand.NewSource(time.Now().UnixNano())),
		Stay: false,
	}

}

//Update update yeelight info
func (y *Yeelight) Update() bool {

	// if !y.Connected || y.Socket == nil {
	// 	y.connect()
	// 	if y.Socket == nil {
	// 		return false
	// 	}
	// }

	on, err := y.GetProp("power", "color_mode", "ct", "rgb", "hue", "sat", "bright", "flowing", "delayoff", "flow_params", "music_on")
	if err != nil {
		if strings.Contains(err.Error(), "i/o timeout") {
			y.disconnect()
		}
		fmt.Println(err)
		fmt.Println("err Update: " + y.Addr + " + " + y.ID)
		return false
	}
	fmt.Println("UPDATE " + y.Addr)

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

	go func() {
		if y.ID != "" {
			newData := sdk.Data{
				Plugin:       Config.Name,
				PhysicalName: y.Model,
				PhysicalID:   y.ID,
			}

			for _, field := range sdk.FindDevicesFromName(Config.Devices, y.Model).Triggers {
				saveValue := reflect.ValueOf(y).Elem().FieldByName(field.Name).String()
				if field.Type == "int" {
					saveValue = strconv.Itoa(int(reflect.ValueOf(y).Elem().FieldByName(field.Name).Int()))
				}
				newData.Values = append(newData.Values, sdk.Value{
					Name:  field.Name,
					Value: []byte(saveValue),
					Type:  field.Type,
				})
			}
			datas <- newData
		}
	}()

	return true
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

//StartFunc is used to Launch every yeelight action.
func (y *Yeelight) StartFunc(funcName string, values ...interface{}) error {
	_, err := y.executeCommand(funcName, values...)

	return err
}

// GetProp method is used to retrieve current property of smart LED.
func (y *Yeelight) GetProp(values ...interface{}) ([]interface{}, error) {
	r, err := y.executeCommand("get_prop", values...)
	if nil != err {
		return nil, err
	}
	if r == nil || r.Result == nil {
		return nil, errors.New("no data found")
	}
	return r.Result, nil
}

//executeCommand executes command with provided parameters
func (y *Yeelight) executeCommand(name string, params ...interface{}) (*CommandResult, error) {
	return y.execute(&Command{
		Method: name,
		ID:     y.randID(),
		Params: params,
	})
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

	b, err := json.Marshal(cmd)
	if err != nil || y.Socket == nil {
		fmt.Println(err)
		return nil, nil
	}
	fmt.Fprint(y.Socket, string(b)+crlf)

	reply := make([]byte, 1024)
	if y.Socket == nil {
		return nil, nil
	}
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

	return &rs, nil

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
