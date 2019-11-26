package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

//StartFunc is used to Launch every yeelight action.
func (y *Yeelight) StartFunc(funcName string, values ...interface{}) (interface{}, error) {
	r, err := y.executeCommand(funcName, values...)
	if nil != err {
		return nil, err
	}
	if r == nil || r.Result == nil {
		return nil, errors.New("no data found")
	}
	return r.Result, nil
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
	var rs CommandResult

	hasError := true
	for hasError {
		if !y.Connected || y.Socket == nil {
			hasError = true
			y.connect()
			fmt.Println("Reconnect yeelight if no connection")

			continue
		}

		y.Socket.SetReadDeadline(time.Now().Add(10 * time.Second))

		b, err := json.Marshal(cmd)
		if err != nil || y.Socket == nil {
			hasError = true
			fmt.Println("No socket 1")
			continue
		}
		fmt.Fprint(y.Socket, string(b)+crlf)

		if y.Socket == nil {
			hasError = true
			fmt.Println("No socket 2")
			continue
		}
		reply, err := bufio.NewReader(y.Socket).ReadString('\n')
		if err != nil {
			fmt.Println(fmt.Errorf("cannot read command result %s", err))
			if strings.Contains(err.Error(), "EOF") {
				y.disconnect()
			}
			if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "i/o timeout") {
				hasError = false
				break
			}
			hasError = true
			continue
		}

		err = json.Unmarshal([]byte(reply), &rs)
		if nil != err {
			hasError = true
			fmt.Println(fmt.Errorf("cannot parse command result %s", err))
			continue
		}
		if nil != rs.Error {
			fmt.Println(fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message))
			if strings.Contains(rs.Error.Message, "client quota exceeded") || rs.Error.Code == -5001 {
				hasError = false
				break
			}

			hasError = true
			continue
		}

		hasError = false
	}
	return &rs, nil
}
