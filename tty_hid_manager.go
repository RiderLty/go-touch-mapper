package main

import (
	"encoding/binary"

	"go.bug.st/serial"
)

func rot_xy(pack touch_control_pack) (int32, int32) { //根据方向旋转坐标
	switch global_device_orientation {
	case 0:
		return pack.x, pack.y
	case 1:
		return pack.screen_y - pack.y, pack.x
	case 2:
		return pack.screen_x - pack.x, pack.screen_y - pack.y
	case 3:
		return pack.y, pack.screen_x - pack.x
	default:
		return pack.x, pack.y
	}
}

func handel_touch_using_hid_manager(port serial.Port, screenX, screenY uint32, rotation int) touch_control_func {

	var buf [12]byte
	buf[0] = 0xF4
	setReport := func(action uint8, id uint8, x, y uint32) {
		buf[1] = action
		buf[2] = id
		binary.LittleEndian.PutUint32(buf[3:7], x)
		binary.LittleEndian.PutUint32(buf[7:11], y)
		buf[11] = 0
	}

	rot_xy := func(pack touch_control_pack) (int32, int32) { //根据方向旋转坐标
		switch rotation {
		case 0:
			return pack.x, pack.y
		case 1:
			return pack.screen_y - pack.y, pack.x
		case 2:
			return pack.screen_x - pack.x, pack.screen_y - pack.y
		case 3:
			return pack.y, pack.screen_x - pack.x
		default:
			return pack.x, pack.y
		}
	}

	return func(control_data touch_control_pack) {
		switch control_data.action {

		// case TouchActionRequire or TouchActionMove:
		case TouchActionRequire, TouchActionMove:
			x, y := rot_xy(control_data)
			setReport(0x01, uint8(control_data.id), uint32(x), uint32(y))
			port.Write(buf[:])
		case TouchActionRelease:
			setReport(0x00, uint8(control_data.id), 0, 0)
			port.Write(buf[:])
		case TouchActionResetResolution:
			setReport(0x03, uint8(control_data.id), uint32(control_data.x), uint32(control_data.y))
			port.Write(buf[:])
		}
	}
}
