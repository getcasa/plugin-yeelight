package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
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

	if !y.Connected || y.Socket == nil {
		y.connect()
		return nil, nil
	}

	y.Socket.SetReadDeadline(time.Now().Add(timeout))

	b, err := json.Marshal(cmd)
	if err != nil || y.Socket == nil {
		fmt.Println(err)
		return nil, nil
	}
	fmt.Fprint(y.Socket, string(b)+crlf)

	// reply := make([]byte, 1024)
	if y.Socket == nil {
		return nil, nil
	}
	// size, err := y.Socket.Read(reply)
	reply, err := bufio.NewReader(y.Socket).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("cannot read command result %s", err)
	}

	// reply = reply[:size]

	var rs CommandResult
	err = json.Unmarshal([]byte(reply), &rs)
	if nil != err {
		return nil, fmt.Errorf("cannot parse command result %s", err)
	}
	if nil != rs.Error {
		return nil, fmt.Errorf("command execution error. Code: %d, Message: %s", rs.Error.Code, rs.Error.Message)
	}

	return &rs, nil

}
