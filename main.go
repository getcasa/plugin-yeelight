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
	}
)

// Params define actions parameters available
type Params struct {
	State bool
}

// OnStart start UDP server to get Xiaomi data
func OnStart(config []byte) {
	go Discover()

	go func() {
		for range time.Tick(5 * time.Second) {
			findIDLight()
		}
	}()

	return
}

// CallAction call functions from actions
func CallAction(physicalID string, name string, params []byte, config []byte) {
	if string(params) == "" {
		fmt.Println("Params must be provided")
		return
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
	default:
	}
	yee.Update()
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

	err := discover()
	go findIDLight()
	if err != nil {
		fmt.Println(err)
		return Discover()
	}
	for _, light := range lights {
		if light.ID == "" {
			continue
		}

		devices = append(devices, sdk.Device{
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
				if findLightWithAddr(ip+":"+strconv.Itoa(yeelightPort)) != nil {
					wg.Done()
					return
				}
				ps := portscanner.NewPortScanner(ip, 10*time.Second, 4)
				opened := ps.IsOpen(yeelightPort)
				if !opened {
					wg.Done()
					return
				}
				newyee := New(ip + ":" + strconv.Itoa(yeelightPort))
				newyee.connect()
				if newyee == nil || !newyee.Connected {
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

func findIDLight() {
	var check bool
	check = true
	for check {
		check = false
		for _, light := range lights {
			if light.ID == "" {
				check = true
				break
			}
		}
		ssdp, _ := net.ResolveUDPAddr("udp4", ssdpAddr)
		c, _ := net.ListenPacket("udp4", ":0")
		socket := c.(*net.UDPConn)
		socket.WriteToUDP([]byte(discoverMSG), ssdp)
		socket.SetReadDeadline(time.Now().Add(timeout))

		rsBuf := make([]byte, 1024)
		size, _, err := socket.ReadFromUDP(rsBuf)
		if err != nil {
			fmt.Println(err)
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

func (y *Yeelight) stayActive() {
	for range time.Tick(20 * time.Second) {
		y.Update()
	}
}

func (y *Yeelight) connect() {
	var err error
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

	info, err := y.GetProp("model", "name")
	if err == nil {
		if len(info) >= 1 {
			y.Model = info[0].(string)
		}
		if len(info) >= 2 {
			y.Name = info[1].(string)
		}
	}

	// go y.stayActive()
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
	if err != nil {
		return false
	}

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

// GetProp method is used to retrieve current property of smart LED.
func (y *Yeelight) GetProp(values ...interface{}) ([]interface{}, error) {
	r, err := y.executeCommand("get_prop", values...)
	if nil != err {
		return nil, err
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

	b, _ := json.Marshal(cmd)
	fmt.Fprint(y.Socket, string(b)+crlf)

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
