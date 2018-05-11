package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type BasicMsg struct {
	Typ byte    // 1字节整数
	Uid [5]byte // 5字节整数
	Act byte    // 1字节整数
}

// 0-重写; 4-清空; 5-撤销
func rcuMsg(act byte) (bts []byte, err error) {
	msg := &BasicMsg{
		Typ: 1,
		Act: act,
	}

	var uid int64 = 129
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}

	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

// 注册指令
type RegisterMsg struct {
	BasicMsg
	Width  [2]byte // 2字节整数
	Height [2]byte // 2字节整数
}

func registerMsg() (bts []byte, err error) {
	var msg RegisterMsg
	msg.Typ = 1
	msg.Act = 1

	var uid int64 = 129
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	var width int32 = 100
	if err = binary.Write(buf, binary.BigEndian, width); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Width[k] = v
	}
	buf.Reset()

	var height int32 = 50
	if err = binary.Write(buf, binary.BigEndian, height); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Height[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}

	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

type UpdateMsg struct {
	BasicMsg
	R    byte // 2字节整数
	G    byte // 2字节整数
	B    byte // 1字节整数
	Size byte // 1字节整数
}

func updateMsg() (bts []byte, err error) {
	var msg UpdateMsg
	msg.Typ = 1
	msg.Act = 2
	msg.R = 0
	msg.G = 0
	msg.B = 0
	msg.Size = 2

	var uid int64 = 129
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}
	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

type CoordinateMsg struct {
	BasicMsg
	X        [2]byte
	Y        [2]byte
	Pressure byte
	State    byte
}

func coordinateMsg() (bts []byte, err error) {
	var msg CoordinateMsg
	msg.Typ = 1
	msg.Act = 3
	msg.Pressure = 9
	msg.State = 0

	buf := new(bytes.Buffer)

	var uid int64 = 129
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	var x int32 = 10
	if err = binary.Write(buf, binary.BigEndian, x); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.X[k] = v
	}
	buf.Reset()

	var y int32 = 10
	if err = binary.Write(buf, binary.BigEndian, y); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Y[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}
	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

func main() {

	var act byte = 0
	rcuMsg(act)
	registerMsg()
	updateMsg()
	coordinateMsg()

	return

	var n byte = 1
	//var pi float64 = math.Pi
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, n)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	fmt.Printf("% x\n", buf.Bytes())
}
