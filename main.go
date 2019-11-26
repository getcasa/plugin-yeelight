package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	Discover:    true,
	Devices: []sdk.Device{
		sdk.Device{
			Name:           "stripe",
			Description:    "",
			DefaultTrigger: "Power",
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
			Actions: []string{"toggle", "set_power", "set_ct", "set_rgb", "set_hsv", "set_bright", "set_default", "start_cf", "stop_cf", "cron_add", "cron_del", "set_adjust", "set_music"},
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
			Actions: []string{"toggle", "set_power", "set_ct", "set_rgb", "set_hsv", "set_bright", "set_default", "start_cf", "stop_cf", "cron_add", "cron_del", "set_adjust", "set_music"},
		},
	},
	Actions: []sdk.Action{
		sdk.Action{
			Name:   "toggle",
			Fields: []sdk.Field{},
		},
		sdk.Action{
			Name: "set_power",
			Fields: []sdk.Field{
				sdk.Field{
					Name:          "power",
					Type:          "string",
					Possibilities: []string{"off", "on"},
					Config:        true,
				},
				sdk.Field{
					Name:          "effect",
					Type:          "string",
					Possibilities: []string{"off", "on"},
					Config:        true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Min:    30,
					Config: true,
				},
				sdk.Field{
					Name:   "mode",
					Type:   "int",
					Min:    0,
					Max:    5,
					Config: true,
				},
			},
		},
		sdk.Action{
			Name: "set_ct",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "ct",
					Type:   "int",
					Config: true,
				},
				sdk.Field{
					Name:          "effect",
					Type:          "string",
					Possibilities: []string{"sudden", "smooth"},
					Config:        true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Min:    30,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_rgb",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "rgb_value",
					Type:   "int",
					Min:    0,
					Max:    16777215,
					Config: true,
				},
				sdk.Field{
					Name:          "effect",
					Type:          "string",
					Possibilities: []string{"sudden", "smooth"},
					Config:        true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Min:    30,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_hsv",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "hue",
					Type:   "int",
					Min:    0,
					Max:    359,
					Config: true,
				},
				sdk.Field{
					Name:   "sat",
					Type:   "int",
					Min:    0,
					Max:    100,
					Config: true,
				},
				sdk.Field{
					Name:          "effect",
					Type:          "string",
					Possibilities: []string{"sudden", "smooth"},
					Config:        true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Min:    30,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_bright",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "brightness",
					Type:   "int",
					Min:    1,
					Max:    100,
					Config: true,
				},
				sdk.Field{
					Name:          "effect",
					Type:          "string",
					Possibilities: []string{"sudden", "smooth"},
					Config:        true,
				},
				sdk.Field{
					Name:   "duration",
					Type:   "int",
					Min:    30,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_default",
			Fields: []sdk.Field{
				sdk.Field{},
			},
		},

		sdk.Action{
			Name: "start_cf",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "count",
					Type:   "int",
					Min:    0,
					Config: true,
				},
				sdk.Field{
					Name:   "action",
					Type:   "int",
					Min:    0,
					Max:    2,
					Config: true,
				},
				sdk.Field{
					Name:   "flow_expression",
					Type:   "string",
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "stop_cf",
			Fields: []sdk.Field{
				sdk.Field{},
			},
		},

		sdk.Action{
			Name: "cron_add",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "type",
					Type:   "int",
					Min:    0,
					Max:    0,
					Config: true,
				},
				sdk.Field{
					Name:   "value",
					Type:   "int",
					Min:    1,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "cron_del",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "type",
					Type:   "int",
					Min:    0,
					Max:    0,
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_adjust",
			Fields: []sdk.Field{
				sdk.Field{
					Name:          "action",
					Type:          "string",
					Possibilities: []string{"increase", "decrease", "circle"},
					Config:        true,
				},
				sdk.Field{
					Name:          "prop",
					Type:          "string",
					Possibilities: []string{"bright", "ct", "color"},
					Config:        true,
				},
			},
		},

		sdk.Action{
			Name: "set_music",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "action",
					Type:   "int",
					Min:    0,
					Max:    1,
					Config: true,
				},
				sdk.Field{
					Name:   "host",
					Type:   "string",
					Config: true,
				},
				sdk.Field{
					Name:   "post",
					Type:   "string",
					Config: true,
				},
			},
		},

		sdk.Action{
			Name: "set_name",
			Fields: []sdk.Field{
				sdk.Field{
					Name:   "name",
					Type:   "string",
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
	Power          string           `db:"power" json:"power"`
	Ct             int              `db:"ct" json:"ct"`
	RGBValue       int              `db:"rgb_value" json:"rgbValue"`
	Effect         string           `db:"effect" json:"effect"`
	Duration       int              `db:"duration" json:"duration"`
	Action         int              `db:"action" json:"action"`
	ActionAdjust   string           `db:"action_adjust" json:"actionAdjust"`
	Hue            int              `db:"hue" json:"hue"`
	Sat            int              `db:"sat" json:"sat"`
	Bright         int              `db:"bright" json:"bright"`
	Count          int              `db:"count" json:"count"`
	FlowExpression []flowExpression `db:"flow_expression" json:"flowExpression"`
	Mode           int              `db:"mode" json:"mode"`
	Host           string           `db:"host" json:"host"`
	Port           string           `db:"port" json:"port"`
	Type           string           `db:"type" json:"type"`
	Prop           string           `db:"prop" json:"prop"`
	ActionStr      string           `db:"action_str" json:"actionStr"`
	Value          string           `db:"value" json:"value"`
	Name           string           `db:"name" json:"name"`
}

type flowExpression struct {
	Duration int
	Mode     int
	Value    int
	Bright   int
}

// OnStart start UDP server to get Xiaomi data
func OnStart(config []byte) {
	datas = make(chan sdk.Data)
	go findIDLight()

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
	case "toggle":
		_, err := yee.StartFunc("toggle")
		if err == nil {
			go yee.UpdateToggle()
		}
	case "set_power":
		_, err := yee.StartFunc("set_power", req.Power, req.Effect, req.Duration, req.Mode)
		if err == nil {
			go yee.Update(true)
		}
	case "set_ct":
		_, err := yee.StartFunc("set_ct_abx", req.Ct, req.Effect, req.Duration)
		if err == nil {
			go yee.Update(true)
		}
	case "set_rgb":
		_, err := yee.StartFunc("set_rgb", req.RGBValue, req.Effect, req.Duration)
		if err == nil {
			go yee.Update(true)
		}
	case "set_hsv":
		_, err := yee.StartFunc("set_rgb", req.Hue, req.Sat, req.Effect, req.Duration)
		if err == nil {
			go yee.Update(true)
		}
	case "set_bright":
		_, err := yee.StartFunc("set_bright", req.Bright, req.Effect, req.Duration)
		if err == nil {
			go yee.Update(true)
		}
	case "set_default":
		_, err := yee.StartFunc("set_default")
		if err == nil {
			go yee.Update(true)
		}
	case "start_cf":
		flowExpression := "{"
		for ind, flow := range req.FlowExpression {
			flowExpression += strconv.Itoa(flow.Duration) + "," + strconv.Itoa(flow.Mode) + "," + strconv.Itoa(flow.Value) + "," + strconv.Itoa(flow.Bright)
			if ind != len(req.FlowExpression)-1 {
				flowExpression += ","
			}
		}
		flowExpression += "}"
		_, err := yee.StartFunc("start_cf", req.Count, req.Action, flowExpression)
		if err == nil {
			go yee.Update(true)
		}
	case "stop_cf":
		_, err := yee.StartFunc("stop_cf")
		if err == nil {
			go yee.Update(true)
		}
	case "cron_add":
		_, err := yee.StartFunc("cron_add", req.Type, req.Value)
		if err == nil {
			go yee.Update(true)
		}
	case "cron_del":
		_, err := yee.StartFunc("cron_del", req.Type)
		if err == nil {
			go yee.Update(true)
		}
	case "set_adjust":
		_, err := yee.StartFunc("set_adjust", req.ActionAdjust, req.Prop)
		if err == nil {
			go yee.Update(true)
		}
	case "set_music":
		_, err := yee.StartFunc("set_music", req.Action, req.Host, req.Port)
		if err == nil {
			go yee.Update(true)
		}
	case "set_name":
		_, err := yee.StartFunc("set_name", req.Name)
		if err == nil {
			go yee.Update(true)
		}
	default:
	}
}

// OnStop close connection
func OnStop() {
	for _, light := range lights {
		light.disconnect()
	}
	lights = nil
}

//Discover discovers device in local network
func Discover() []sdk.DiscoveredDevice {
	var devices []sdk.DiscoveredDevice

	go findIDLight()

	for _, light := range lights {
		light.Update(true)
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

func findIDLight() {
	ssdp, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	c, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		fmt.Println(err)
	}
	socket := c.(*net.UDPConn)
	for start := time.Now(); time.Since(start) < 1*time.Minute; {
		socket.SetWriteDeadline(time.Now().Add(timeout))
		socket.WriteToUDP([]byte(discoverMSG), ssdp)
		socket.SetReadDeadline(time.Now().Add(timeout))

		rsBuf := make([]byte, 1024)
		size, _, err := socket.ReadFromUDP(rsBuf)
		if err != nil {
			fmt.Println(err)
			c, err = net.ListenPacket("udp4", ":0")
			socket = c.(*net.UDPConn)
			continue
		}
		rs := rsBuf[0:size]

		addr := parseAddr(string(rs))
		newyee := findLightWithAddr(addr)
		if newyee == nil {
			newyee = New(addr)
			lights = append(lights, newyee)
			newyee.connect()
			fmt.Println(addr + " - " + parseID(string(rs)))
		}
		newyee.ID = parseID(string(rs))
		newyee.Model = parseInfo(string(rs), "Model")
		newyee.Name = parseInfo(string(rs), "Name")
		newyee.Power = parseInfo(string(rs), "Power")
		newyee.Bright, _ = strconv.Atoi(parseInfo(string(rs), "Bright"))
		newyee.ColorMode, _ = strconv.Atoi(parseInfo(string(rs), "Color_mode"))
		newyee.CT, _ = strconv.Atoi(parseInfo(string(rs), "CT"))
		newyee.RGB, _ = strconv.Atoi(parseInfo(string(rs), "RGB"))
		newyee.Hue, _ = strconv.Atoi(parseInfo(string(rs), "Hue"))
		newyee.Sat, _ = strconv.Atoi(parseInfo(string(rs), "Sat"))
	}
	err = socket.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func (y *Yeelight) connect() {
	var err error
	if y.Socket != nil {
		y.disconnect()
	}
	y.Socket, err = net.Dial("tcp4", y.Addr)
	if err != nil {
		fmt.Println("Error connection: " + y.Addr + ", error: " + err.Error())
		return
	}

	fmt.Println("Connected: " + y.Addr)
	y.Connected = true

	if !y.Stay {
		y.Stay = true
		go y.stayActive()
	}
}

func (y *Yeelight) stayActive() {
	count := 0
	for range time.Tick(10 * time.Second) {
		if y.Connected {
			count++
			if count%6 == 0 {
				y.Update(true)
			}
			y.Update(false)
		}
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
func (y *Yeelight) Update(saveState bool) bool {

	on, err := y.GetProp("power", "color_mode", "ct", "rgb", "hue", "sat", "bright", "flowing", "delayoff", "flow_params", "music_on")
	if err != nil {
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

	if saveState {
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
	}

	return true
}

//UpdateToggle update toggle yeelight info
func (y *Yeelight) UpdateToggle() bool {

	on, err := y.GetProp("power")
	if err != nil || on[0].(string) == "ok" {
		fmt.Println(err)
		fmt.Println("err Update Toggle: " + y.Addr + " + " + y.ID)
		return false
	}

	if len(on) >= 1 {
		y.Power = on[0].(string)
	}

	go func() {
		if y.ID != "" {
			newData := sdk.Data{
				Plugin:       Config.Name,
				PhysicalName: y.Model,
				PhysicalID:   y.ID,
			}

			newData.Values = append(newData.Values, sdk.Value{
				Name:  "Power",
				Value: []byte(y.Power),
				Type:  "string",
			})

			datas <- newData
		}
	}()

	return true
}

//closeConnection closes network connection
func closeConnection(c net.Conn) {
	if nil != c {
		c.Close()
	}
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

//parseInfo parses address from ssdp response
func parseInfo(msg string, field string) string {
	if strings.HasSuffix(msg, crlf) {
		msg = msg + crlf
	}
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(msg)), nil)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get(field)
}

func (y *Yeelight) randID() int {
	i := y.rnd.Intn(100)
	return i
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
