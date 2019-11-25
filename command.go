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

	fmt.Println(cmd.Params)

	hasError := true
	for hasError {
		fmt.Println("wow !")

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
			if strings.Contains(err.Error(), "EOF") {
				hasError = false
				break
			}
			hasError = true
			fmt.Println(fmt.Errorf("cannot read command result %s", err))
			continue
			return nil, fmt.Errorf("cannot read command result %s", err)
		}

		err = json.Unmarshal([]byte(reply), &rs)
		if nil != err {
			hasError = true
			fmt.Println(fmt.Errorf("cannot parse command result %s", err))
			continue
			return nil, fmt.Errorf("cannot parse command result %s", err)
		}
		if nil != rs.Error {
			if strings.Contains(rs.Error.Message, "client quota exceeded") || rs.Error.Code == -5001 {
				hasError = false
				break
			}

			hasError = true
			fmt.Println(fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message))
			continue
			return nil, fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message)
		}

		hasError = false
		continue
	}
	return &rs, nil
}
