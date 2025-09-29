package main

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"os"
	"unsafe"

	"github.com/lunixbochs/struc"
	"golang.org/x/sys/unix"

	"github.com/kenshaw/evdev"
)

func toUInputName(name []byte) [uinputMaxNameSize]byte {
	var fixedSizeName [uinputMaxNameSize]byte
	copy(fixedSizeName[:], name)
	return fixedSizeName
}

func uInputDevToBytes(uiDev UinputUserDev) []byte {
	var buf bytes.Buffer
	_ = struc.PackWithOptions(&buf, &uiDev, &struc.Options{Order: binary.LittleEndian})
	return buf.Bytes()
}

func createDevice(f *os.File) (err error) {
	return ioctl(f.Fd(), UIDEVCREATE(), uintptr(0))
}

func create_u_input_touch_screen(width int32, height int32) *os.File {
	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		logger.Errorf("create u_input touch_screen error:%v", err)
		return nil
	}
	ioctl(deviceFile.Fd(), UISETEVBIT(), evKey)
	ioctl(deviceFile.Fd(), UISETKEYBIT(), 0x014a) //一个是BTN_TOUCH 一个不知道是啥
	ioctl(deviceFile.Fd(), UISETKEYBIT(), 0x003e) //是从手机直接copy出来的

	ioctl(deviceFile.Fd(), UISETEVBIT(), evAbs)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtSlot)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtTrackingId)

	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtTouchMajor)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtWidthMajor)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtPositionX)
	ioctl(deviceFile.Fd(), UISETABSBIT(), absMtPositionY)

	ioctl(deviceFile.Fd(), UISETPROPBIT(), inputPropDirect)

	var absMin [absCnt]int32
	absMin[absMtPositionX] = 0
	absMin[absMtPositionY] = 0
	absMin[absMtTouchMajor] = 0
	absMin[absMtWidthMajor] = 0
	absMin[absMtSlot] = 0
	absMin[absMtTrackingId] = 0

	var absMax [absCnt]int32
	absMax[absMtPositionX] = width << touch_pos_scale  //可以通过缩放 来获得更高精度
	absMax[absMtPositionY] = height << touch_pos_scale //坐标系会自动以原点缩放
	absMax[absMtTouchMajor] = 255
	absMax[absMtWidthMajor] = 0
	absMax[absMtSlot] = 9
	absMax[absMtTrackingId] = 65535

	uiDev := UinputUserDev{
		Name: toUInputName([]byte("v_touch_screen")),
		ID: InputID{
			BusType: 0,
			Vendor:  randUInt16Num(0x2000),
			Product: randUInt16Num(0x2000),
			Version: randUInt16Num(0x20),
		},
		EffectsMax: 0,
		AbsMax:     absMax,
		AbsMin:     absMin,
		AbsFuzz:    [absCnt]int32{},
		AbsFlat:    [absCnt]int32{},
	}
	deviceFile.Write(uInputDevToBytes(uiDev))
	createDevice(deviceFile)
	return deviceFile
}

// func create_u_input_touch_screen(index int) *os.File {
// 	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
// 	if err != nil {
// 		logger.Errorf("读取设备失败 : %v", err)
// 		return nil
// 	}
// 	d := evdev.Open(fd)
// 	defer d.Close()

// 	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
// 	if err != nil {
// 		logger.Errorf("create u_input touch_screen error:%v", err)
// 		return nil
// 	}

// 	ioctl(deviceFile.Fd(), UISETEVBIT(), evKey)
// 	for key := range d.KeyTypes() {
// 		ioctl(deviceFile.Fd(), UISETKEYBIT(), uintptr(key)) //一个是BTN_TOUCH 一个不知道是啥
// 	}

// 	ioctl(deviceFile.Fd(), UISETEVBIT(), evAbs)
// 	var initAbsVal [absCnt]int32
// 	var initAbsMin [absCnt]int32
// 	var initAbsMax [absCnt]int32
// 	var initAbsFuzz [absCnt]int32
// 	var initAbsFlat [absCnt]int32
// 	var initAbsRes [absCnt]int32

// 	for key, value := range d.AbsoluteTypes() {
// 		ioctl(deviceFile.Fd(), UISETABSBIT(), uintptr(key))
// 		initAbsVal[key] = value.Val
// 		initAbsMin[key] = value.Min
// 		initAbsMax[key] = value.Max
// 		initAbsFuzz[key] = value.Fuzz
// 		initAbsFlat[key] = value.Flat
// 		initAbsRes[key] = value.Res
// 	}

// 	ioctl(deviceFile.Fd(), UISETPROPBIT(), inputPropDirect)

// 	uiDev := UinputUserDev{
// 		Name: toUInputName([]byte("v_touch_screen_KKK")),
// 		ID: InputID{
// 			BusType: 0,
// 			Vendor:  randUInt16Num(0x2000),
// 			Product: randUInt16Num(0x2000),
// 			Version: randUInt16Num(0x20),
// 		},
// 		EffectsMax: 0,
// 		AbsMax:     initAbsMax,
// 		AbsMin:     initAbsMin,
// 		AbsFuzz:    initAbsFlat,
// 		AbsFlat:    initAbsFlat,
// 	}
// 	deviceFile.Write(uInputDevToBytes(uiDev))
// 	createDevice(deviceFile)
// 	return deviceFile
// }

func create_u_input_mouse_keyboard() *os.File {
	deviceFile, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		logger.Errorf("create u_input mouse error:%v", err)
		return nil
	}
	ioctl(deviceFile.Fd(), UISETEVBIT(), evSyn)
	ioctl(deviceFile.Fd(), UISETEVBIT(), evKey)
	ioctl(deviceFile.Fd(), UISETEVBIT(), evRel)
	ioctl(deviceFile.Fd(), UISETEVRELBIT(), relX)
	ioctl(deviceFile.Fd(), UISETEVRELBIT(), relY)
	ioctl(deviceFile.Fd(), UISETEVRELBIT(), relWheel)
	ioctl(deviceFile.Fd(), UISETEVRELBIT(), relHWheel)
	for i := 0x110; i < 0x117; i++ {
		ioctl(deviceFile.Fd(), UISETKEYBIT(), uintptr(i))
	}
	for i := 0; i < 256; i++ {
		ioctl(deviceFile.Fd(), UISETKEYBIT(), uintptr(i))
	}

	uiDev := UinputUserDev{
		Name: toUInputName([]byte("go-touch-mapper-virtual-device")),
		ID: InputID{
			BusType: 0,
			Vendor:  randUInt16Num(0x2000),
			Product: randUInt16Num(0x2000),
			Version: randUInt16Num(0x20),
		},
		EffectsMax: 0,
		AbsMax:     [absCnt]int32{},
		AbsMin:     [absCnt]int32{},
		AbsFuzz:    [absCnt]int32{},
		AbsFlat:    [absCnt]int32{},
	}
	deviceFile.Write(uInputDevToBytes(uiDev))
	createDevice(deviceFile)
	return deviceFile
}

func handel_u_input_mouse_keyboard(u_input chan *u_input_control_pack) {
	sizeofEvent := int(unsafe.Sizeof(evdev.Event{}))
	sendEvents := func(fd *os.File, events []*evdev.Event) {
		if fd == nil {
			logger.Warnf("fd is nil,pass %v", events)
			return
		}

		buf := make([]byte, sizeofEvent*len(events))
		for i, event := range events {
			copy(buf[i*sizeofEvent:], (*(*[1<<27 - 1]byte)(unsafe.Pointer(event)))[:sizeofEvent])
		}
		n, err := fd.Write(buf)
		if err != nil {
			logger.Errorf("write %v bytes error:%v", n, err)
		}
	}
	ev_sync := evdev.Event{Type: EV_SYN, Code: 0, Value: 0}
	fd := create_u_input_mouse_keyboard()
	for {
		write_events := make([]*evdev.Event, 0)
		select {
		case <-global_close_signal:
			return
		case pack := <-u_input:
			switch pack.action {
			case UInput_mouse_move:
				write_events = append(write_events, &evdev.Event{Type: EV_REL, Code: REL_X, Value: pack.arg1})
				write_events = append(write_events, &evdev.Event{Type: EV_REL, Code: REL_Y, Value: pack.arg2})
				write_events = append(write_events, &ev_sync)
				sendEvents(fd, write_events)
			case UInput_mouse_btn:
				write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: uint16(pack.arg1), Value: pack.arg2})
				write_events = append(write_events, &ev_sync)
				sendEvents(fd, write_events)
			case UInput_mouse_wheel:
				write_events = append(write_events, &evdev.Event{Type: EV_REL, Code: uint16(pack.arg1), Value: pack.arg2})
				write_events = append(write_events, &ev_sync)
				sendEvents(fd, write_events)
			case UInput_key_event:
				write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: uint16(pack.arg1), Value: pack.arg2})
				write_events = append(write_events, &ev_sync)
				sendEvents(fd, write_events)
			}
		}
	}
}

const (
	ABS_MT_POSITION_X  = 0x35
	ABS_MT_POSITION_Y  = 0x36
	ABS_MT_SLOT        = 0x2F
	ABS_MT_TRACKING_ID = 0x39
	EV_SYN             = 0x00
	EV_KEY             = 0x01
	EV_REL             = 0x02
	EV_ABS             = 0x03
	REL_X              = 0x00
	REL_Y              = 0x01
	REL_WHEEL          = 0x08
	REL_HWHEEL         = 0x06
	SYN_REPORT         = 0x00
	BTN_TOUCH          = 0x14A
)

func get_wm_size() (int32, int32) {
	cmd := exec.Command("sh", "-c", "wm size")
	out, err := cmd.Output()
	if err != nil {
		logger.Errorf("get wm size error:%v", err)
		os.Exit(4)
	}
	wxh := strings.TrimSpace(strings.Split(strings.ReplaceAll(string(out), "\n", " "), " ")[2])
	res := strings.Split(wxh, "x")
	width, _ := strconv.Atoi(res[0])
	height, _ := strconv.Atoi(res[1])
	return int32(width), int32(height)
}

type EventMap struct {
	data   []byte
	Events []evdev.Event
}

func makeEventsMMap(size int) EventMap {
	sizeofEvent := int(unsafe.Sizeof(evdev.Event{}))
	var byteSlice = make([]byte, size)
	dataPtr := unsafe.Pointer(&byteSlice[0])
	eventSlice := unsafe.Slice((*evdev.Event)(dataPtr), size/sizeofEvent)
	// return byteSlice, eventSlice
	return EventMap{data: byteSlice, Events: eventSlice}
}

func handel_touch_using_uinput_touch() touch_control_func {
	// sizeofEvent := int(unsafe.Sizeof(evdev.Event{}))
	// sendEvents := func(fd *os.File, events []*evdev.Event) {
	// 	buf := make([]byte, sizeofEvent*len(events))
	// 	for i, event := range events {
	// 		copy(buf[i*sizeofEvent:], (*(*[1<<27 - 1]byte)(unsafe.Pointer(event)))[:sizeofEvent])
	// 	}
	// 	start := time.Now()
	// 	logger.Debugf("Bytes {%s}", hex.EncodeToString(buf))
	// 	// n, err := fd.Write(buf)

	// 	unix.Write(int(fd.Fd()), buf)

	// 	logger.Debugf("sendEvents write %v bytes using %v", len(events)*24, time.Since(start))
	// 	// if err != nil {
	// 	// 	logger.Errorf("handel_touch_using_vTouch error on writing %v bytes :%v", n, err)
	// 	// }

	// }
	rot_xy := func(pack touch_control_pack) (int32, int32) { //根据方向旋转坐标
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
	// ev_sync := evdev.Event{Type: EV_SYN, Code: 0, Value: 0}
	var count int32 = 0    //BTN_TOUCH 申请时为1 则按下 释放时为0 则松开
	var last_id int32 = -1 //ABS_MT_SLOT last_id每次动作后修改 如果不等则额外发送MT_SLOT事件
	w, h := get_wm_size()
	logger.Infof("已创建虚拟触屏 : %vx%v", w, h)
	fd := create_u_input_touch_screen(w, h)
	unixFd := int(fd.Fd())
	// logger.Infof("已从/dev/input/event%d创建虚拟触屏", index)
	// fd := create_u_input_touch_screen(index)

	// nfd, _ := os.OpenFile("/dev/null", syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	// fd = nfd
	go func() {
		<-global_close_signal
		fd.Close()
	}()

	// return func(control_data touch_control_pack) {
	// 	// start := time.Now()
	// 	write_events := make([]*evdev.Event, 0)
	// 	if control_data.id == -1 { //在任何正常情况下 这里是拿不到ID=-1的控制包的因此可以直接丢弃
	// 		return
	// 	}
	// 	if control_data.action == TouchActionRequire {
	// 		last_id = control_data.id
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: control_data.id})
	// 		count += 1
	// 		if count == 1 {
	// 			write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: DOWN})
	// 		}
	// 		x, y := rot_xy(control_data)
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: x})
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: y})
	// 		write_events = append(write_events, &ev_sync)
	// 		sendEvents(fd, write_events)
	// 	} else if control_data.action == TouchActionRelease {
	// 		if last_id != control_data.id {
	// 			last_id = control_data.id
	// 			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
	// 		}
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1})
	// 		count -= 1
	// 		if count == 0 {
	// 			write_events = append(write_events, &evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: UP})
	// 		}
	// 		write_events = append(write_events, &ev_sync)
	// 		sendEvents(fd, write_events)
	// 	} else if control_data.action == TouchActionMove {
	// 		if last_id != control_data.id {
	// 			last_id = control_data.id
	// 			write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: control_data.id})
	// 		}
	// 		x, y := rot_xy(control_data)
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: x})
	// 		write_events = append(write_events, &evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: y})
	// 		write_events = append(write_events, &ev_sync)
	// 		sendEvents(fd, write_events)
	// 	}
	// 	// logger.Debugf("uinput handeler%v", time.Since(start))
	// }

	require_init := makeEventsMMap(6 * 24)
	require := makeEventsMMap(5 * 24)

	release := makeEventsMMap(2 * 24)
	switch_release := makeEventsMMap(3 * 24)
	switch_release_btnup := makeEventsMMap(4 * 24)
	release_btnup := makeEventsMMap(3 * 24)

	switch_move := makeEventsMMap(4 * 24)
	move := makeEventsMMap(3 * 24)

	require_init.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0}
	require_init.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: 0}
	require_init.Events[2] = evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: DOWN}
	require_init.Events[3] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: 0}
	require_init.Events[4] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: 0}
	require_init.Events[5] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	require.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0}
	require.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: 0}
	require.Events[2] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: 0}
	require.Events[3] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: 0}
	require.Events[4] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	release.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1}
	release.Events[1] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	switch_release.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0}
	switch_release.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1}
	switch_release.Events[2] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	switch_release_btnup.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0}
	switch_release_btnup.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1}
	switch_release_btnup.Events[2] = evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: UP}
	switch_release_btnup.Events[3] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	release_btnup.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1}
	release_btnup.Events[1] = evdev.Event{Type: EV_KEY, Code: BTN_TOUCH, Value: UP}
	release_btnup.Events[2] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	switch_move.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0}
	switch_move.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: 0}
	switch_move.Events[2] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: 0}
	switch_move.Events[3] = evdev.Event{Type: EV_SYN, Code: 0, Value: 0}

	move.Events[0] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: 0}
	move.Events[1] = evdev.Event{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: 0}

	// type pack struct {
	// 	data []byte
	// 	ts   time.Time
	// }
	// packChan := make(chan pack, 1)
	// go (func() {
	// 	for {
	// 		select {
	// 		case tmp := <-packChan:
	// 			unix.Write(unixFd, tmp.data)
	// 			logger.Debugf("using chan %v", time.Since(tmp.ts))
	// 		default:
	// 			unix.Write(unixFd, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	// 		}
	// 	}
	// })()

	return func(control_data touch_control_pack) {
		// write_events := make([]*evdev.Event, 0)
		// start := time.Now()
		if control_data.id == -1 { //在任何正常情况下 这里是拿不到ID=-1的控制包的因此可以直接丢弃
			return
		}
		if control_data.action == TouchActionRequire {
			x, y := rot_xy(control_data)
			last_id = control_data.id
			if count += 1; count == 1 {
				require_init.Events[0].Value = control_data.id
				require_init.Events[1].Value = control_data.id
				require_init.Events[3].Value = x
				require_init.Events[4].Value = y
				unix.Write(unixFd, require_init.data)
				// packChan <- pack{data: require_init.data, ts: start}
			} else {
				require.Events[0].Value = control_data.id
				require.Events[1].Value = control_data.id
				require.Events[2].Value = x
				require.Events[3].Value = y
				unix.Write(unixFd, require.data)
				// packChan <- pack{data: require.data, ts: start}

			}

		} else if control_data.action == TouchActionRelease {
			if last_id != control_data.id {
				last_id = control_data.id
				if count -= 1; count == 0 {
					switch_release_btnup.Events[0].Value = control_data.id
					unix.Write(unixFd, switch_release_btnup.data)
					// packChan <- pack{data: switch_release_btnup.data, ts: start}
				} else {
					switch_release.Events[0].Value = control_data.id
					unix.Write(unixFd, switch_release.data)
					// packChan <- pack{data: switch_release.data, ts: start}
				}
			} else {
				if count -= 1; count == 0 {
					unix.Write(unixFd, release_btnup.data)
					// packChan <- pack{data: release_btnup.data, ts: start}
				} else {
					unix.Write(unixFd, release.data)
					// packChan <- pack{data: release.data, ts: start}
				}
			}
		} else if control_data.action == TouchActionMove {
			x, y := rot_xy(control_data)
			if last_id != control_data.id {
				last_id = control_data.id
				switch_move.Events[0].Value = control_data.id
				switch_move.Events[1].Value = x
				switch_move.Events[2].Value = y
				unix.Write(unixFd, switch_move.data)
				// packChan <- pack{data: switch_move.data, ts: start}

			} else {
				move.Events[0].Value = x
				move.Events[1].Value = y
				unix.Write(unixFd, move.data)
				// packChan <- pack{data: move.data, ts: start}
			}
		}
		// logger.Debugf("uinput handeler%v", time.Since(start))
	}
}

//
//func handelTouchUsingEsp32() touch_control_func {
//
//	rot_xy := func(pack touch_control_pack) (int32, int32) { //根据方向旋转坐标
//		switch global_device_orientation {
//		case 0:
//			return pack.x, pack.y
//		case 1:
//			return pack.screen_y - pack.y, pack.x
//		case 2:
//			return pack.screen_x - pack.x, pack.screen_y - pack.y
//		case 3:
//			return pack.y, pack.screen_x - pack.x
//		default:
//			return pack.x, pack.y
//		}
//	}
//
//	mode := &serial.Mode{
//		BaudRate: 2000000,
//	}
//	port, err := serial.Open("/dev/ttyACM0", mode)
//	if err != nil {
//		panic(err)
//	}
//
//	go func() {
//		<-global_close_signal
//		port.Close()
//	}()
//	// 创建触摸报告数据包
//	createTouchReport := func(id byte, x, y int32, down bool) []byte {
//		if x < 0 {
//			x = 0
//		}
//		if y < 0 {
//			y = 0
//		}
//		if x > 119999 { // 32bit X 范围
//			x = 119999
//		}
//		if y > 266999 { // 32bit Y 范围
//			y = 266999
//		}
//
//		state := byte(0x01) // 按下
//		if !down {
//			state = 0x00 // 抬起
//		}
//
//		// 创建小端序字节切片（10 字节）
//		data := make([]byte, 12)
//		data[0] = state
//		data[1] = id
//		binary.LittleEndian.PutUint32(data[4:8], uint32(x))
//		binary.LittleEndian.PutUint32(data[8:12], uint32(y))
//
//		return data
//	}
//
//	return func(control_data touch_control_pack) {
//		// write_events := make([]*evdev.Event, 0)
//		// start := time.Now()
//		if control_data.id == -1 { //在任何正常情况下 这里是拿不到ID=-1的控制包的因此可以直接丢弃
//			return
//		}
//		if control_data.action == TouchActionRequire {
//			data := createTouchReport(byte(control_data.id), control_data.x, control_data.y, true)
//
//			_, _ = port.Write(data)
//
//		} else if control_data.action == TouchActionRelease {
//			data := createTouchReport(byte(control_data.id), control_data.x, control_data.y, false)
//
//			_, _ = port.Write(data)
//
//		} else if control_data.action == TouchActionMove {
//			data := createTouchReport(byte(control_data.id), control_data.x, control_data.y, true)
//			_, _ = port.Write(data)
//		}
//		// logger.Debugf("uinput handeler%v", time.Since(start))
//	}
//}
