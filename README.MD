# 安卓触屏映射工具

读取手柄，键盘，鼠标，触摸屏等输入并转换为虚拟触屏设备输出

[视频教程](https://www.bilibili.com/video/BV1XtgeztEvY/)

## Quick start

从release直接下载编译好的二进制程序

一般手机平板下载 [go-touch-mapper_arm64](https://github.com/RiderLty/go-touch-mapper/releases/latest/download/go-touch-mapper_arm64)

androidX86下载 [go-touch-mapper_amd64](https://github.com/RiderLty/go-touch-mapper/releases/latest/download/go-touch-mapper_amd64)

或者使用国内镜像

一般手机平板 [go-touch-mapper_arm64](https://vip.123pan.cn/1833788059/direct/projects/go-touch-mapper/build/go-touch-mapper_arm64)

androidX86 [go-touch-mapper_amd64](https://vip.123pan.cn/1833788059/direct/projects/go-touch-mapper/build/go-touch-mapper_amd64)

读取dev和创建uinput需要shell或者root权限

使用adb shell或者su shell运行

```
cd /data/local/tmp && \ 
curl -LJo go-touch-mapper https://vip.123pan.cn/1833788059/direct/projects/go-touch-mapper/build/go-touch-mapper_arm64 && \
chmod 777 ./go-touch-mapper && \
./go-touch-mapper -h
```
如果提示无curl，则需要手动下载后push到/data/local/tmp下运行

## Build & Run

可在安卓设备的termux环境下自己编译
```
pkg i golang yarn -y
git clone https://github.com/RiderLty/go-touch-mapper
cd go-touch-mapper
cd go-touch-mapper-gh-pages
yarn install
yarn build
cd ..
CGO_ENABLE=0 GOOS=linux go build -ldflags="-s -w"
./go-touch-mapper -h
```

## Help

```
usage: go-touch-mapper [-h|--help] [-c|--config "<value>"] [--create-js-info]
                       [--pattern "<value>"] [-s|--sender "<value>"] [-m|--mode
                       "<value>"] [-t|--disable-mix] [-u|--disable-uinput]
                       [--display-id <integer>] [--tty-path "<value>"]
                       [--rotation <integer>] [-r|--remote-control] [-p|--port
                       <integer>] [-v|--v-mouse] [--v-mouse-addr "<value>"]
                       [--auto-release <integer>] [--measure-mode]
                       [-d|--debug-mode]

                        

Arguments:

  -h  --help            Print help information
  -c  --config         
                        指定配置文件路径，如果不存在则会使用默认模板创建
      --create-js-info  创建手柄配置文件模式. Default: false
      --pattern         用于筛选设备名称的正则. Default: .*
  -s  --sender          发送本地事件到远程，输入 IP:PORT 例如
                        192.168.3.7:61069
                        或者仅输入IP使用默认端口61069. Default: 
  -m  --mode            触摸方案，可用控制模式:   
                                uinput:         使用uinput创建虚拟触屏 
                                inputmanager:   通过UDS控制安卓inputManager
                                hid:            通过串口控制单片机模拟usb触屏 
                                otg:            本机配置LinuxUSBgadget模拟usb触屏,设备文件为/dev/hidg0
                                direct:         直接写入设备真实触屏,需要root权限或者低版本安卓.
                        Default: uinput
  -t  --disable-mix    
                        关闭触屏混合,仅在uinput与inputmanager模式生效.
                        Default: false
  -u  --disable-uinput 
                        不再创建uinput鼠标键盘设备,仅在uinput、inputmanager与direct模式生效.
                        Default: false
      --display-id     
                        显示器ID,仅inputmanager模式生效,多显示器情况下可控制额外的显示器.
                        Default: 0
      --tty-path        串口设备路径,hid模式下必须指定. Default: 
      --rotation       
                        手动指定屏幕方向,仅在hid与otg模式下需要(建议使用-v).
                        Default: 1
  -r  --remote-control  是否从UDP接收远程事件. Default: false
  -p  --port           
                        指定监听远程事件的UDP端口号与控制后台端口.
                        Default: 61069
  -v  --v-mouse        
                        用触摸操作模拟鼠标,需要额光标外显示程序vPointer,同时可用于在hid与otg模式下获取屏幕方向.
                        Default: false
      --v-mouse-addr   
                        模拟光标显示程序地址,默认本机6533端口,输入IP:PORT,例如192.168.3.7:6533,或者仅输入IP使用默认端口6533.
                        Default: 
      --auto-release   
                        触发视角自动释放所需的静止ms数,50ms为检查单位,置0禁用.
                        Default: 200
      --measure-mode   
                        显示视角移动像素计数,且可输入数值模拟滑动,方向键可微调,用于测试灵敏度.
                        Default: false
  -d  --debug-mode      打印debug信息. Default: false
```
## 运行示例

```
./go-touch-mapper -c df.json
```

如果df.json不存在，会从模板创建配置文件。

## 映射开关
默认键盘使用 ` 键，可在配置文件中手动修改，手柄固定为SELECT+RS组合键

## 配置文件

程序运行后会显示控制后台地址```http://[ip]:61070/```用于编辑配置文件。

如有需要，可参照[EXAMPLE.JSON](https://github.com/RiderLty/go-touch-mapper/blob/main/configs/EXAMPLE.JSON)手动修改配置文件

可自动检测屏幕旋转方向，始终以左上角为原点，向右为X，向下为Y

按住键盘或者手柄按键后，点击屏幕对应位置即可添加按键映射

灵敏度设置 MOUSE:SPEED:[ x , y ] float64类型，此数值也会影响右摇杆控制视角的速度


# 触摸输出模式选择 -m

## uinput

默认参数

使用uinput创建虚拟触屏，也是最推荐的模式

## [InputManager](https://github.com/RiderLty/inputManager-touch-interface)

使用参数```-m inputmanager```启用

由Scrcpy修改而来，使用app_process启动

通过 Unix Domain Socket 与主程序通信控制触屏

在多显示器情况下可以使用```--display-id```指定displayID

只可在adb shell下使用


## HID触屏

使用参数```-m hid```启用

使用ESP32-s3的HID库，创建一个外置触屏，在上位机通过串口通信的方式发送触摸指令

优点在于无需在被控设备上安装任何软件或者执行指令

### 使用方式

* 刷入hid_touch.ino固件   [点击在线烧录](https://riderlty.github.io/go-touch-mapper/)

* 连接设备，进入[配置页面](https://riderlty.github.io/dualsense-tester/)，在未配置的情况下为模拟DS5手柄，可通过短按boot简单测试，长按boot全量测试，长按5s重置。

* DS5模式下，部分安卓设备不识别，强烈建议选择HID模式为自定义模式，然后手动配置硬件信息。写入后记得重启保存。

* 将键鼠手柄与ESP32-s3的串口连接到例如树莓派之类的linux设备上，USB口连接到被控设备

* 执行 ```ls /dev | grep ttyACM``` 查看串口设备

* 使用--tty-path指定串口设备，--rotation 指定旋转方向，例如：```./go-touch-mapper -m hid --tty-path /dev/ttyACM0 ``` 

* 测试回报率在1000hz以下无明显延迟，超过1000hz会出现操作丢失。

* 部分设备会出现输入迟缓的情况，建议降低输入设备的回报率到500hz或者更低。

## OTG模式

使用参数```-m otg```启用

* 树莓派5，安装 [pi-gadget-hid](https://github.com/RiderLty/pi-gadget-hid)

* 通过type-c供电数据分离线，为树莓派供电同时，数据线连接主机

* 使用 --otg 参数 并且手动指定旋转方向 --rotation

* 将键鼠插在树莓派上即可控制

## 真实触屏设备文件

使用参数```-m direct```启用

直接写入设备的真实触屏，需要root权限

搭配```-r -t -u```参数，不创建任何虚拟设备，仅接收UDP事件处理后直接写入真实触屏

⚠ ⚠ ⚠ 绝对不要同时使用映射的同时手指点击屏幕

## 触屏混合

程序会识别并读取机器物理触屏输入，与键鼠混合后使用虚拟触屏输出以实现映射开启时，触摸屏幕不中断操作。

可使用-t禁用此功能

⚠ 如果是在手机端进行配置文件的修改，建议禁用，避免无法操作手机。

## uinput键鼠

会将所有的设备包括远程事件混合后使用uinput创建的键鼠输出。

可使用-u禁用创建uinput键鼠

## [虚拟光标显示程序](https://github.com/RiderLty/vPointer)

从release中下载最新的安装包，安装后授予悬浮窗权限。

运行时添加使用 ```-v``` 参数，将使用触摸操作替代鼠标，仅左键可用作点击。

适用于某些屏蔽了鼠标点击的游戏

使用串口与树莓派otg模式时候，可设置--v-mouse-addr以操控手机端显示光标

此时会忽略 --rotation 指定的方向并自动识别

## 手柄支持
joystickInfos文件夹下json文件为手柄配置文件

按照按键位置统一为XBOX手柄按键布局，可直接下载使用。

扳机键按照力度映射为5个等级的按键 BTN_LT_1 ... BTN_LT_5，等级5同时会出发BTN_LT控制信号。

没有线性扳机的手柄例如 Switch Pro controller 只有 BTN_LT 和 BTN_RT

  注意⚠ : 

* 手柄的按键代码在不同设备上可能有所差异
* 如果出现按键无反应的情况，建议重新创建手柄配置文件

### 手动创建手柄配置文件
```
./go-touch-mapper --create-js-info 
```

## 远程控制

添加 -r 参数

```
./go-touch-mapper -r
```

PC上 修改remote_control中sender("192.168.1.64:61069")为手机IP，端口默认可不修改

运行

``` 
python remote_control.py 
```


或者下载[打包的exe](remote_control/dist/remote_control.exe)

在同目录创建[addr.txt](remote_control/dist/addr.txt)，写入要控制设备的IP:端口后运行

或者使用 -s 参数，将本机变为远程控制发送端

实现在另一台安卓手机或者linux设备上发送控制指令
```
./go-touch-mapper -s 127.0.0.1
```

## 插件系统
目前开放了wasd映射轮盘的接口

编辑plugin/custom_plugin.go，实现自己的逻辑，然后编译

```
cd plugin
CGO_ENABLED=0 GOOS=linux GOARCH=arm64  go build -ldflags="-s -w" -tags plugin -o plugin.bin
```
将获得的plugin.bin放在同目录下即可调用

PluginID用于指定配置文件ID，会创建pluginconfig/${pluginconfig}.json文件存储插件参数

PluginConfigTemplate指定插件需要的参数，会显示在web配置界面




## ADB 下载

[platform-tools-latest-windows](https://vip.123pan.cn/1833788059/23477218)

## 许可证

本项目采用 [MIT License](LICENSE)。版权归 RiderLty 所有，具体条款请参阅仓库根目录的 [LICENSE](LICENSE) 文件。
