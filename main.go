package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/akamensky/argparse"
	"github.com/bytedance/sonic"
	"github.com/kenshaw/evdev"
	"go.bug.st/serial"
	"log"
	"math"
)

type event_pack struct {
	//表示一个动作 由一系列event组成
	dev_name string
	dev_type dev_type
	events   []*evdev.Event
}

type touch_control_pack struct {
	//触屏控制信息
	action   int8
	id       int32
	x        int32
	y        int32
	screen_x int32
	screen_y int32
}

type u_input_control_pack struct {
	action int8
	arg1   int32
	arg2   int32
}

type touch_control_func func(data touch_control_pack)

func dev_reader(event_reader chan *event_pack, index int) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		logger.Errorf("读取设备失败 : %v", err)
		return
	}
	d := evdev.Open(fd)
	defer d.Close()
	event_ch := d.Poll(context.Background())
	events := make([]*evdev.Event, 0)
	dev_name := d.Name()
	logger.Infof("开始读取设备 : %s", dev_name)
	d.Lock()
	defer d.Unlock()
	for {
		select {
		case <-global_close_signal:
			logger.Infof("释放设备 : %s", dev_name)
			return
		case event := <-event_ch:
			if event == nil {
				logger.Warnf("移除设备 : %s", dev_name)
				return
			} else if event.Type == evdev.SyncReport {
				pack := &event_pack{
					dev_name: dev_name,
					dev_type: check_dev_type(d),
					events:   events,
				}
				event_reader <- pack
				events = make([]*evdev.Event, 0)
			} else {
				events = append(events, &event.Event)
			}
		}
	}
}

func udp_event_injector(ch chan *event_pack, port int) {
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	})
	if err != nil {
		logger.Errorf("udp error : %v", err)
		return
	}
	defer listen.Close()

	recv_ch := make(chan []byte)
	go func() {
		for {
			var buf [1024]byte
			n, _, err := listen.ReadFromUDP(buf[:])
			if err != nil {
				break
			}
			recv_ch <- buf[:n]
		}
	}()
	logger.Infof("已准备接收远程事件: 0.0.0.0:%d", port)
	for {
		select {
		case <-global_close_signal:
			return
		case pack := <-recv_ch: //数据包格式：<event_count:1byte><event1:8byte><event2:8byte>...<eventN:8byte><dev_type:1byte><dev_name:N byte>
			//每个event格式：<type:2byte><code:2byte><value:4byte>
			//共8byte
			// logger.Debugf("%v", pack)
			event_count := int(pack[0])
			events := make([]*evdev.Event, 0)
			for i := 0; i < event_count; i++ {
				event := &evdev.Event{
					Type:  evdev.EventType(uint16(binary.LittleEndian.Uint16(pack[8*i+1 : 8*i+3]))),
					Code:  uint16(binary.LittleEndian.Uint16(pack[8*i+3 : 8*i+5])),
					Value: int32(binary.LittleEndian.Uint32(pack[8*i+5 : 8*i+9])),
				}
				// logger.Debugf("%v", event)
				events = append(events, event)
			}
			e_pack := &event_pack{
				dev_name: string(pack[event_count*8+2:]),
				dev_type: dev_type(pack[event_count*8+1]),
				events:   events,
			}
			ch <- e_pack
			// logger.Debugf("接收到事件 : %v", e_pack)
		}
	}
}
func unixEventInjector(ch chan *event_pack, port int) {
	listen, err := net.Listen("unix", "/tmp/control.sock")
	if err != nil {
		logger.Errorf("udp error : %v", err)
		return
	}
	defer listen.Close()
	//listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", port))
	//if err != nil {
	//	logger.Errorf("udp error : %v", err)
	//	return
	//}
	recv_ch := make(chan []byte)
	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				logger.Errorf("Accept error: %v", err)
				break
			}

			// 为每个连接启动一个处理 goroutine
			go func(c net.Conn) {
				defer c.Close()

				// 使用缓冲区来处理粘包问题
				var buf []byte
				readBuf := make([]byte, 40960) // 读取缓冲区
				expectedLength := -1           // 期望的消息长度，-1表示需要读取包头

				for {
					n, err := c.Read(readBuf)
					if err != nil {
						logger.Errorf("Read error: %v", err)
						break
					}

					// 将新读取的数据添加到缓冲区
					buf = append(buf, readBuf[:n]...)

					// 处理缓冲区中的所有完整消息
					for {
						// 如果还不知道消息长度，先读取4字节包头
						if expectedLength == -1 && len(buf) >= 4 {
							expectedLength = int(binary.BigEndian.Uint32(buf[:4]))
							buf = buf[4:] // 移除包头
						}

						// 如果已经知道消息长度且缓冲区中有足够数据
						if expectedLength != -1 && len(buf) >= expectedLength {
							// 提取完整消息
							message := buf[:expectedLength]
							recv_ch <- message

							// 移除已处理的数据
							buf = buf[expectedLength:]
							expectedLength = -1 // 重置为需要读取下一个包头

							// 继续处理缓冲区中可能存在的下一条消息
							continue
						}

						// 没有足够数据或没有完整消息，跳出内层循环继续读取
						break
					}
				}
			}(conn)
		}
	}()

	logger.Infof("已准备接收远程事件: 0.0.0.0:%d", port)
	for {
		select {
		case <-global_close_signal:
			return
		case pack := <-recv_ch:
			if len(pack) < 1 {
				logger.Warnf("收到空数据包")
				continue
			}

			var eventPack struct {
				//表示一个动作 由一系列event组成
				DevName string         `json:"dev_name"`
				DevType dev_type       `json:"dev_type"`
				Events  []*evdev.Event `json:"events"`
			}
			t := time.Now()
			if err = sonic.Unmarshal(pack, &eventPack); err != nil {
				continue
			}
			logger.Debugf("解码时间: %v", time.Since(t))
			ePack := &event_pack{
				dev_name: eventPack.DevName,
				dev_type: eventPack.DevType,
				events:   eventPack.Events,
			}
			ch <- ePack
		}
	}
}

var global_close_signal = make(chan bool) //仅会在程序退出时关闭  不用于其他用途
var global_device_orientation int32 = 0

func get_device_orientation() int32 {
	output, err := exec.Command("sh", "-c", "dumpsys input").Output()
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`orientation=(\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		orientation := matches[1]
		result, err := strconv.Atoi(string(orientation))
		if err != nil {
			return 0
		} else {
			return int32(result)
		}
	} else {
		return 0
	}
}

func listen_device_orientation() {
	for {
		select {
		case <-global_close_signal:
			return
		default:
			var now_orientation int32 = get_device_orientation()
			if global_device_orientation != now_orientation {
				global_device_orientation = now_orientation
				logger.Debugf("设备方向改变\t[%d]", now_orientation)
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

type dev_type uint8

const (
	type_mouse    = dev_type(0)
	type_keyboard = dev_type(1)
	type_joystick = dev_type(2)
	type_touch    = dev_type(3)
	type_unknown  = dev_type(4)
)

func check_dev_type(dev *evdev.Evdev) dev_type {
	abs := dev.AbsoluteTypes()
	key := dev.KeyTypes()
	rel := dev.RelativeTypes()
	_, MTPositionX := abs[evdev.AbsoluteMTPositionX]
	_, MTPositionY := abs[evdev.AbsoluteMTPositionY]
	_, MTSlot := abs[evdev.AbsoluteMTSlot]
	_, MTTrackingID := abs[evdev.AbsoluteMTTrackingID]
	if MTPositionX && MTPositionY && MTSlot && MTTrackingID {
		return type_touch //触屏检测这几个abs类型即可
	}
	_, RelX := rel[evdev.RelativeX]
	_, RelY := rel[evdev.RelativeY]
	_, HWheel := rel[evdev.RelativeHWheel]
	_, MouseLeft := key[evdev.BtnLeft]
	_, MouseRight := key[evdev.BtnRight]
	if RelX && RelY && HWheel && MouseLeft && MouseRight {
		return type_mouse //鼠标 检测XY 滚轮 左右中键
	}
	keyboard_keys := true
	for i := evdev.KeyEscape; i <= evdev.KeyScrollLock; i++ {
		_, ok := key[i]
		keyboard_keys = keyboard_keys && ok
	}
	if keyboard_keys {
		return type_keyboard //键盘 检测keycode(1-70)
	}

	axis_count := 0
	for i := evdev.AbsoluteX; i <= evdev.AbsoluteRZ; i++ {
		_, ok := abs[i]
		if ok {
			axis_count++
		}
	}
	if axis_count >= 4 { //检测有两个摇杆
		return type_joystick
	}
	return type_unknown
}

func get_possible_device_indexes(skipList map[int]bool) map[int]dev_type {
	files, _ := ioutil.ReadDir("/dev/input")
	result := make(map[int]dev_type)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if len(file.Name()) <= 5 {
			continue
		}
		if file.Name()[:5] != "event" {
			continue
		}
		index, _ := strconv.Atoi(file.Name()[5:])
		reading, exist := skipList[index]
		if exist && reading {
			continue
		} else {
			fd, err := os.OpenFile(fmt.Sprintf("/dev/input/%s", file.Name()), os.O_RDONLY, 0)
			if err != nil {
				logger.Errorf("读取设备/dev/input/%s失败 : %v ", file.Name(), err)
			}
			d := evdev.Open(fd)
			defer d.Close()
			devType := check_dev_type(d)
			if devType != type_unknown {
				result[index] = devType
			}
		}
	}
	return result
}

func get_dev_name_by_index(index int) string {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
	if err != nil {
		return "读取设备名称失败"
	}
	d := evdev.Open(fd)
	defer d.Close()
	return d.Name()
}

// func get_dev_phys_info_by_index(index int) string {
// 	fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
// 	if err != nil {
// 		return "read name error"
// 	}
// 	d := evdev.Open(fd)
// 	defer d.Close()
// 	return d.Name()
// 	// return d
// }

func execute_view_move(handelerInstance *TouchHandler, x, stepValue, sleepMS int) {
	handelerInstance.handel_view_move(0, 0)
	time.Sleep(time.Millisecond * time.Duration(sleepMS))
	if x > 0 {
		steps := x / stepValue
		for i := 0; i < steps; i++ {
			if !handelerInstance.map_on {
				break
			}
			handelerInstance.handel_view_move(int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(int32(x%stepValue), 0)
	} else {
		steps := -x / stepValue
		for i := 0; i < steps; i++ {
			if !handelerInstance.map_on {
				break
			}
			handelerInstance.handel_view_move(-int32(stepValue), 0)
			time.Sleep(time.Millisecond * time.Duration(sleepMS))
		}
		handelerInstance.handel_view_move(-int32(x%stepValue), 0)
	}
}

func stdin_control_view_move(handelerInstance *TouchHandler) {
	logger.Info("输入数值以精确控制view移动 [ x | x 步长 间隔 ]")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		var x, stepValue, sleepMS int
		var err error
		if len(args) == 1 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			} else {
				stepValue = 24
				sleepMS = 16
			}

		} else if len(args) == 3 {
			x, err = strconv.Atoi(args[0])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			stepValue, err = strconv.Atoi(args[1])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
			sleepMS, err = strconv.Atoi(args[2])
			if err != nil {
				logger.Errorf("输入错误: %s", err)
				continue
			}
		} else {
			logger.Error("参数错误 usage:x | x stepValue sleepMS")
			continue
		}
		logger.Infof("x: %d stepValue: %d sleepMS: %d", x, stepValue, sleepMS)
		if handelerInstance.map_on {
			execute_view_move(handelerInstance, x, stepValue, sleepMS)
		} else {
			logger.Info("等待映射开关打开中...")
			for {
				if !handelerInstance.map_on {
					time.Sleep(time.Duration(100) * time.Millisecond)
				} else {
					execute_view_move(handelerInstance, x, stepValue, sleepMS)
					break
				}
			}
		}
	}
}

func get_MT_size(indexes map[int]bool) (int32, int32) { //获取MTPositionX和MTPositionY的max值
	//在1+7p上是等于get_wm_size
	//但是红魔7sp应该是为了更精确，使用了缩放,实际的数值为 rawValue >> 3
	// rawValue * wm_size / mt_size = true_value
	for index, _ := range indexes {
		fd, err := os.OpenFile(fmt.Sprintf("/dev/input/event%d", index), os.O_RDONLY, 0)
		if err != nil {
			logger.Errorf("get_MT_size error:%v", err)
		}
		d := evdev.Open(fd)
		defer d.Close()
		abs := d.AbsoluteTypes()
		MTPositionX, _ := abs[evdev.AbsoluteMTPositionX]
		MTPositionY, _ := abs[evdev.AbsoluteMTPositionY]
		return MTPositionX.Max, MTPositionY.Max
	}
	return int32(1), int32(1)
}

func OpenSerialWritePipe(portName string, baudRate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}
	return port, nil
}

func auto_detect_and_read(event_chan chan *event_pack, patern string) {
	//自动检测设备并读取 循环检测 自动管理设备插入移除
	devices := make(map[int]bool)
	for {
		select {
		case <-global_close_signal:
			return
		default:
			auto_detect_result := get_possible_device_indexes(devices)
			devTypeFriendlyName := map[dev_type]string{
				type_mouse:    "鼠标",
				type_keyboard: "键盘",
				type_joystick: "手柄",
				type_touch:    "触屏",
				type_unknown:  "未知",
			}
			for index, devType := range auto_detect_result {
				devName := get_dev_name_by_index(index)
				if devName == "go-touch-mapper-virtual-device" {
					continue //跳过生成的虚拟设备
				}
				re := regexp.MustCompile(patern)
				if !re.MatchString(devName) {
					// logger.Debugf("设备名称 %s 不匹配 %s", devName, patern)
					continue
				}
				if devType == type_mouse || devType == type_keyboard || devType == type_joystick {
					logger.Infof("检测到设备 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
					localIndex := index
					go func() {
						devices[localIndex] = true
						dev_reader(event_chan, localIndex)
						devices[localIndex] = false
					}()
				}
			}
			time.Sleep(time.Duration(400) * time.Millisecond)
		}
	}
}

func parseHIDTouchScreen(s string) (x, y uint32, r int, err error) {
	parts := strings.Split(s, "x")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid format: expected 3 parts separated by 'x', got %d", len(parts))
	}
	x64, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid x value: %v", err)
	}
	x = uint32(x64)
	y64, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid y value: %v", err)
	}
	y = uint32(y64)
	r, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid r value: %v", err)
	}
	if r < 0 || r > 3 {
		return 0, 0, 0, fmt.Errorf("r value must be 0, 1, 2, or 3, got %d", r)
	}
	return x, y, r, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

//go:embed configs
var configs embed.FS

func main() {
	parser := argparse.NewParser("go-touch-mapper", " ")

	var configPath *string = parser.String("c", "config", &argparse.Options{
		Required: false,
		Help:     "指定配置文件路径，如果不存在则会使用默认模板创建",
	})

	var create_js_info *bool = parser.Flag("", "create-js-info", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "创建手柄配置文件模式",
	})

	var patern *string = parser.String("", "pattern", &argparse.Options{
		Required: false,
		Default:  ".*",
		Help:     "用于筛选设备名称的正则",
	})

	var mixTouchDisabled *bool = parser.Flag("t", "touch-disabled", &argparse.Options{
		Required: false,
		Help:     "关闭触屏混合",
		Default:  false,
	})

	var usingInputManagerID *int = parser.Int("i", "inputManager", &argparse.Options{
		Required: false,
		Default:  -1,
		Help:     "DisplayID 使用inputManager控制触摸，可用于控制外接显示器 ",
	})

	var usingHIDTouchTtyPath *string = parser.String("", "tty-path", &argparse.Options{
		Required: false,
		Default:  "",
		Help:     "串口设备路径 将使用串口控制外接的HID设备模拟触屏",
	})

	var usingHIDTouchScreen *string = parser.String("", "tty-screen", &argparse.Options{
		Required: false,
		Default:  "1440x3440x1",
		Help:     "使用串口控制HID设备模拟触屏的屏幕参数，宽x高x旋转方向，例如1440x3440x1",
	})
	var using_remote_control *bool = parser.Flag("r", "remoteControl", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "是否从UDP接收远程事件",
	})

	var port *int = parser.Int("p", "port", &argparse.Options{
		Required: false,
		Help:     "指定监听远程事件的UDP端口号与控制后台端口",
		Default:  61069,
	})

	var using_v_mouse *bool = parser.Flag("v", "v-mouse", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "用触摸操作模拟鼠标,需要额光标外显示程序",
	})

	var view_release_timeout *int = parser.Int("", "auto-release", &argparse.Options{
		Required: false,
		Help:     "触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用",
		Default:  200,
	})

	var measure_sensitivity_mode *bool = parser.Flag("", "measure-mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调",
	})

	var debug_mode *bool = parser.Flag("d", "debug_mode", &argparse.Options{
		Required: false,
		Default:  false,
		Help:     "打印debug信息",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if *usingInputManagerID != -1 && *usingHIDTouchTtyPath != "" {
		logger.Error("无法同时使用inputManager与HID")
		os.Exit(2)
	}

	hid_x, hid_y, hid_r, err := parseHIDTouchScreen(*usingHIDTouchScreen)
	if err != nil {
		logger.Error("解析失败: %v\n", err)
		return
	}

	if *debug_mode {
		logger.WithDebug()
		logger.Debug("debug on")
	}

	if *configPath == "" {
		logger.Warn("未指定配置文件，使用默认配置文件")
		exePath, err := os.Executable()
		if err != nil {
			fmt.Println("获取可执行文件路径失败:", err)
			return
		}
		exeDir := filepath.Dir(exePath)
		*configPath = filepath.Join(exeDir, "default.json")
	}
	if !fileExists(*configPath) {
		bytes, err := configs.ReadFile("configs/EXAMPLE.JSON")
		if err != nil {
			logger.Infof("读取内置配置文件configs/EXAMPLE.JSON错误:%v", err)
			return
		}

		logger.Infof("已从模板创建配置文件:%v", *configPath)

		os.WriteFile(*configPath, bytes, 0644)

	}

	logger.Infof("使用配置文件: %v", *configPath)

	if *create_js_info {
		auto_detect_result := get_possible_device_indexes(make(map[int]bool))
		devTypeFriendlyName := map[dev_type]string{
			type_mouse:    "鼠标",
			type_keyboard: "键盘",
			type_joystick: "手柄",
			type_touch:    "触屏",
			type_unknown:  "未知",
		}
		for index, devType := range auto_detect_result {
			devName := get_dev_name_by_index(index)
			logger.Infof("检测到设备 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
		}
		js_events := make([]int, 0)
		for index, devType := range auto_detect_result {
			if devType == type_joystick {
				js_events = append(js_events, index)
			}
		}
		if len(js_events) == 1 {
			create_js_info_file(js_events[0])
		} else {
			if len(js_events) == 0 {
				logger.Warn("未检测到手柄")
			} else {
				logger.Warn("检测到多个手柄,断开其他手柄的连接")
			}
		}
	} else {
		events_ch := make(chan *event_pack) //主要设备事件管道

		u_input_control_ch := make(chan *u_input_control_pack)
		fileted_u_input_control_ch := make(chan *u_input_control_pack)
		touch_event_ch := make(chan *event_pack)
		max_mt_x, max_mt_y := int32(1), int32(1)

		if !*mixTouchDisabled {
			devTypeFriendlyName := map[dev_type]string{
				type_mouse:    "鼠标",
				type_keyboard: "键盘",
				type_joystick: "手柄",
				type_touch:    "触屏",
				type_unknown:  "未知",
			}
			for index, devType := range get_possible_device_indexes(make(map[int]bool)) {
				if devType == type_touch {
					devName := get_dev_name_by_index(index)
					logger.Infof("启用触屏混合 %s(/dev/input/event%d) : %s", devName, index, devTypeFriendlyName[devType])
					go dev_reader(touch_event_ch, index)
					max_mt_x, max_mt_y = get_MT_size(map[int]bool{index: true})
					break
				}
			}
		}

		go auto_detect_and_read(events_ch, *patern)

		go handel_u_input_mouse_keyboard(fileted_u_input_control_ch)

		var touch_control_func touch_control_func

		if *usingHIDTouchTtyPath != "" {
			logger.Info("触屏控制将使用串口控制外接的HID设备处理")
			logger.Infof("串口路径：%s", *usingHIDTouchTtyPath)
			logger.Infof("HID触屏尺寸：%dx%d", hid_x, hid_y)
			logger.Infof("HID触屏方向：%d", hid_r)

			port, err := OpenSerialWritePipe(*usingHIDTouchTtyPath, 2000000)
			if err != nil {
				logger.Errorf("无法打开串口: %v", err)
				return
			}
			touch_control_func = handel_touch_using_hid_manager(port, hid_x, hid_y, hid_r)
			touch_control_func(touch_control_pack{
				action:   TouchActionResetResolution,
				id:       0,
				x:        int32(hid_x),
				y:        int32(hid_y),
				screen_x: int32(hid_x),
				screen_y: int32(hid_y),
			})
			go func() {
				buffer := make([]byte, 1024)
				dataBuffer := make([]byte, 0, 1024)
				for {
					// 读取数据
					n, err := port.Read(buffer)
					if err != nil {
						log.Println("Read error:", err)
						continue
					}

					if n > 0 {
						dataBuffer = append(dataBuffer, buffer[:n]...)

						// 处理数据包
						packets := parsePackets(&dataBuffer)

						for _, packet := range packets {
							x, y := constrainToInscribedCircleInt(int(packet.X), int(packet.Y), 1023)
							e_pack := &event_pack{
								dev_name: "joy",
								dev_type: 2,
								events: []*evdev.Event{
									{
										Type:  evdev.EventAbsolute,
										Code:  0,
										Value: int32(x),
									},
									{
										Type:  evdev.EventAbsolute,
										Code:  1,
										Value: int32(y),
									},
								},
							}
							events_ch <- e_pack
							fmt.Printf("X: %4d, Y: %4d\n", packet.X, packet.Y)
						}
					}

					time.Sleep(10 * time.Millisecond)
				}
			}()

		} else if *usingInputManagerID != -1 {
			logger.Info("触屏控制将使用inputManager处理")
			go listen_device_orientation()
			touch_control_func = handel_touch_using_input_manager(*usingInputManagerID)
		} else {
			touch_control_func = handel_touch_using_uinput_touch()
			go listen_device_orientation()
		}

		map_switch_signal := make(chan bool) //通知虚拟鼠标当前为鼠标还是映射模式
		touchHandler := InitTouchHandler(
			*configPath,
			events_ch,
			touch_control_func,
			u_input_control_ch,
			*usingInputManagerID == -1,
			map_switch_signal,
			*measure_sensitivity_mode,
		)
		if *usingHIDTouchTtyPath == "" {
			go touchHandler.mix_touch(touch_event_ch, max_mt_x, max_mt_y)
		}
		go touchHandler.auto_handel_view_release(*view_release_timeout)
		go touchHandler.loop_handel_wasd_wheel()
		go touchHandler.loop_handel_rs_move()
		go touchHandler.handel_event()
		if *using_v_mouse {
			v_mouse := init_v_mouse_controller(touchHandler, u_input_control_ch, fileted_u_input_control_ch, map_switch_signal)
			go v_mouse.main_loop()
		} else {
			go (func() {
				for {
					select {
					case tmp := <-u_input_control_ch:
						fileted_u_input_control_ch <- tmp
					case <-map_switch_signal:
					}
				}
			})()
		}

		if *using_remote_control {
			go udp_event_injector(events_ch, *port)
			go unixEventInjector(events_ch, *port)
		}

		if *measure_sensitivity_mode {
			go stdin_control_view_move(touchHandler)
		}

		go serve(*port, *configPath, touchHandler.reloadConfigure) //启动服务器

		exitChan := make(chan os.Signal)
		signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-exitChan
		close(global_close_signal)
		logger.Info("已停止")
		time.Sleep(time.Millisecond * 40)
	}
}

// 解析数据包
func parsePackets(buffer *[]byte) []JoystickData {
	var packets []JoystickData
	data := *buffer

	for {
		// 查找起始字节 0xAA
		startIndex := -1
		for i := 0; i < len(data); i++ {
			if data[i] == 0xAA {
				startIndex = i
				break
			}
		}

		if startIndex == -1 {
			// 没有找到起始字节，清空缓冲区
			*buffer = []byte{}
			return packets
		}

		// 移除起始字节之前的数据
		data = data[startIndex:]

		// 检查是否有足够的数据（6字节：起始标志 + X(2) + Y(2) + 结束标志）
		if len(data) < 6 {
			*buffer = data
			return packets
		}

		// 检查结束字节
		if data[5] != 0x55 {
			// 结束字节不匹配，跳过这个起始字节继续查找
			data = data[1:]
			continue
		}

		// 解析数据
		x := int16(binary.BigEndian.Uint16(data[1:3]))
		y := int16(binary.BigEndian.Uint16(data[3:5]))

		packets = append(packets, JoystickData{X: x, Y: y})

		// 移除已处理的数据包
		data = data[6:]
	}
}

// 整数版本的限位函数（避免浮点运算）
func constrainToInscribedCircleInt(x, y, maxValue int) (int, int) {
	if x == 0 && y == 0 {
		return 0, 0
	}

	// 使用整数运算计算距离平方
	xSq := x * x
	ySq := y * y
	maxSq := maxValue * maxValue

	// 如果坐标点在内接圆外
	if xSq+ySq > maxSq {
		// 计算缩放因子（使用定点数运算）
		// 这里使用 1024 作为精度因子
		distance := int(math.Sqrt(float64(xSq + ySq)))
		scale := (maxValue << 10) / distance // 左移10位相当于乘以1024

		// 应用缩放
		x = (x * scale) >> 10
		y = (y * scale) >> 10
	}

	return x, y
}

type JoystickData struct {
	X int16
	Y int16
}
