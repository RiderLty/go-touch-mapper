#include "USB.h"
#include "USBHID.h"
#include <string.h>

USBHID HID;

// 每个手指使用6字节: 1字节(TipSwitch + 填充), 1字节Contact ID, 2字节X, 2字节Y
#define BYTES_PER_FINGER 6
#define MAX_FINGERS 10
#define DEBUG 1

// 手柄相关定义
#define JOYSTICK_DEADZONE 50      // 死区范围
#define JOYSTICK_READ_INTERVAL 10 // 读取手柄的时间间隔(ms)
#define ADC_MAX 4095              // ESP32-S3 ADC最大读数

// 滤波相关设置
#define FILTER_SAMPLES 5          // 移动平均滤波样本数
#define MEDIAN_FILTER_SIZE 5      // 中值滤波窗口大小
#define FILTER_ALPHA 0.7          // 低通滤波系数

// 全局变量 - 在loop()之外声明
unsigned long lastJoystickRead = 0;

// 滤波数据结构
struct FilterData {
    int16_t samples[FILTER_SAMPLES];
    uint8_t index;
    int32_t sum;
};

FilterData filterX = {0};
FilterData filterY = {0};

// 手柄校准数据结构
struct JoystickCalibration {
  uint16_t centerX, centerY;      // 中心点电压值
  uint16_t minX, maxX;            // X轴最小最大值
  uint16_t minY, maxY;            // Y轴最小最大值
  uint16_t directions[8][2];      // 8个方向的校准点 [方向][x/y]
};

// 全局校准数据
JoystickCalibration joystickCal = {
  .centerX = 1929, .centerY = 1947,  // 默认中心值
  .minX = 0, .maxX = 4095,
  .minY = 0, .maxY = 4095,
  .directions = {
    {4095, 1947}, // 右
    {4095, 4095}, // 右下
    {1927, 4095}, // 下
    {0, 4095},    // 左下
    {0, 1947},    // 左
    {0, 0},       // 左上
    {1929, 0},    // 上
    {4095, 0}     // 右上
  }
};

struct RecTouchReport {
    uint8_t action; // 状态字节（按下/移动/抬起/重置屏幕尺寸）
    uint8_t id;     // 触点ID
    uint32_t x;     // X坐标（小端序）
    uint32_t y;     // Y坐标（小端序）
    uint8_t activeFingers;
};

RecTouchReport data;

static uint8_t report_descriptor[] = {
    0x05, 0x0D, // Usage Page (Digitizer)
    0x09, 0x04, // Usage (Touch Screen)
    0xA1, 0x01, // Collection (Application)
    // Finger 1
    0x09, 0x22, //   Usage (Finger)
    0xA1, 0x02, //   Collection (Logical)
    0x09, 0x42, //     Usage (Tip Switch)
    0x15, 0x00, //     Logical Minimum (0)
    0x25, 0x10, //     Logical Maximum (1)
    0x75, 0x01, //     Report Size (1)
    0x95, 0x01, //     Report Count (1)
    0x81, 0x02, //     Input (Data,Var,Abs)
    0x95, 0x07, //     Report Count (7) - padding to full byte
    0x81, 0x03, //     Input (Const,Var,Abs)
    0x09, 0x51, //     Usage (Contact Identifier)
    0x75, 0x08, //     Report Size (8)
    0x95, 0x01, //     Report Count (1)
    0x81, 0x02, //     Input (Data,Var,Abs)
    0x05, 0x01, //     Usage Page (Generic Desktop)
    // X
    0x09, 0x30,                             // Usage (X)
    0x15, 0x00,                             // Logical Minimum (0)
    0x27, 0x00, 0xB0, 0x04, 0x00, // Logical Maximum (119999)
    0x75, 0x20,                             // Report Size (32)
    0x95, 0x01,                             // Report Count (1)
    0x81, 0x02,                             // Input (Data,Var,Abs)
    // Y
    0x09, 0x31,                             // Usage (Y)
    0x15, 0x00,                             // Logical Minimum (0)
    0x27, 0x00, 0x6E, 0x0A, 0x00, // Logical Maximum (266999)
    0x75, 0x20,                             // Report Size (32)
    0x95, 0x01,
    0x81, 0x02,
    0xC0, //   End Collection
    // Contact Count
    0x05, 0x0D, // Usage Page (Digitizer)
    0x09, 0x54, // Usage (Contact Count)
    0x25, 0x02, // Logical Maximum (2)
    0x75, 0x08, // Report Size (8)
    0x95, 0x01, // Report Count (1)
    0x81, 0x02, // Input (Data,Var,Abs)
    0xC0        // End Collection (Application)
};

class CustomHIDDevice : public USBHIDDevice {
public:
    CustomHIDDevice(void) {
        static bool initialized = false;
        if (!initialized) {
            initialized = true;
            HID.addDevice(this, sizeof(report_descriptor));
        }
    }

    uint16_t _onGetFeature(uint8_t report_id, uint8_t *buffer, uint16_t len) {
        (void)report_id;
        (void)len;
        buffer[0] = 0x0A;
        return 1;
    }

    uint16_t _onGetDescriptor(uint8_t *buffer) {
        memcpy(buffer, report_descriptor, sizeof(report_descriptor));
        return sizeof(report_descriptor);
    }

    void begin(void) {
        HID.begin();
    }

    bool send(uint8_t *value, uint16_t len) {
        return HID.SendReport(0, value, len);
    }
};

CustomHIDDevice Device;

// 读取ADC值（模拟引脚）
uint16_t readADC(uint8_t pin) {
    return analogRead(pin);
}

// 应用死区处理
int16_t applyDeadzone(int16_t value, uint16_t deadzone) {
    if (abs(value) < deadzone) {
        return 0;
    }
    return value;
}

// 移动平均滤波
int16_t movingAverageFilter(FilterData &filter, int16_t newValue) {
    // 减去最旧的值
    filter.sum -= filter.samples[filter.index];
    // 加上新的值
    filter.sum += newValue;
    // 更新样本数组
    filter.samples[filter.index] = newValue;
    // 更新索引
    filter.index = (filter.index + 1) % FILTER_SAMPLES;
    // 返回平均值
    return filter.sum / FILTER_SAMPLES;
}

// 中值滤波
int16_t medianFilter(int16_t newValue) {
    static int16_t medianBuffer[MEDIAN_FILTER_SIZE] = {0};
    static uint8_t medianIndex = 0;

    // 更新缓冲区
    medianBuffer[medianIndex] = newValue;
    medianIndex = (medianIndex + 1) % MEDIAN_FILTER_SIZE;

    // 复制缓冲区用于排序
    int16_t tempBuffer[MEDIAN_FILTER_SIZE];
    memcpy(tempBuffer, medianBuffer, sizeof(tempBuffer));

    // 冒泡排序找中值
    for (uint8_t i = 0; i < MEDIAN_FILTER_SIZE - 1; i++) {
        for (uint8_t j = 0; j < MEDIAN_FILTER_SIZE - i - 1; j++) {
            if (tempBuffer[j] > tempBuffer[j + 1]) {
                int16_t temp = tempBuffer[j];
                tempBuffer[j] = tempBuffer[j + 1];
                tempBuffer[j + 1] = temp;
            }
        }
    }

    return tempBuffer[MEDIAN_FILTER_SIZE / 2];
}

// 带滤波的读取函数
void readJoystickWithFilter(int16_t &x, int16_t &y) {
    // 读取原始ADC值（GPIO8和GPIO9）
    uint16_t rawX = readADC(8);
    uint16_t rawY = readADC(9);

    // 转换为相对中心点的偏移量
    int16_t offsetX = rawX - joystickCal.centerX;
    int16_t offsetY = rawY - joystickCal.centerY;

    // 应用死区
    offsetX = applyDeadzone(offsetX, JOYSTICK_DEADZONE);
    offsetY = applyDeadzone(offsetY, JOYSTICK_DEADZONE);

    // 归一化到-1023到1023范围
    if (offsetX != 0) {
        offsetX = (offsetX > 0) ? map(offsetX, 0, joystickCal.maxX - joystickCal.centerX, 0, 1023)
                                : -map(-offsetX, 0, joystickCal.centerX - joystickCal.minX, 0, 1023);
    }

    if (offsetY != 0) {
        offsetY = (offsetY > 0) ? map(offsetY, 0, joystickCal.maxY - joystickCal.centerY, 0, 1023)
                                : -map(-offsetY, 0, joystickCal.centerY - joystickCal.minY, 0, 1023);
    }

    // 应用组合滤波：先中值后移动平均
    int16_t tempX = medianFilter(offsetX);
    int16_t tempY = medianFilter(offsetY);
    x = movingAverageFilter(filterX, tempX);
    y = movingAverageFilter(filterY, tempY);

    // 最终确保值在范围内
    x = constrain(x, -1023, 1023);
    y = constrain(y, -1023, 1023);
}

// 全局变量保存上一次的值
int16_t lastX = 0;
int16_t lastY = 0;
bool firstRead = true; // 首次读取标志

void outputJoystickState() {
    int16_t x, y;
    readJoystickWithFilter(x, y);

    // 如果是第一次读取或者值有变化才上报
    if (firstRead || x != lastX || y != lastY) {
        // 二进制协议
        uint8_t packet[6] = {
            0xAA,                           // 起始标志
            (uint8_t)((x >> 8) & 0xFF),     // X高字节
            (uint8_t)(x & 0xFF),            // X低字节
            (uint8_t)((y >> 8) & 0xFF),     // Y高字节
            (uint8_t)(y & 0xFF),            // Y低字节
            0x55                            // 结束标志
        };

        Serial.write(packet, 6);

        // 更新上一次的值
        lastX = x;
        lastY = y;
        firstRead = false; // 清除首次读取标志

        // #ifdef DEBUG
        // Serial.print("Filtered Joystick: X=");
        // Serial.print(x);
        // Serial.print(" Y=");
        // Serial.println(y);
        // #endif
    }
}

void setup() {
    Serial.setRxBufferSize(2048); // 将接收缓冲区增加到2KB
    Serial.begin(2000000);
    delay(10);
    Serial.println("ESP32-S3 Dual-touch HID starting...");

    // 初始化ADC引脚
    analogReadResolution(12); // 设置12位分辨率
    analogSetAttenuation(ADC_11db); // 设置衰减器为11dB，量程约0-3.1V

    // 预填充滤波缓冲区（避免初始波动）
    for (int i = 0; i < FILTER_SAMPLES; i++) {
        int16_t rawX = readADC(8) - joystickCal.centerX;
        int16_t rawY = readADC(9) - joystickCal.centerY;
        movingAverageFilter(filterX, rawX);
        movingAverageFilter(filterY, rawY);
        delay(10);
    }

    Device.begin();
    USB.begin();
    unsigned long start = millis();
    while (!HID.ready() && (millis() - start < 3000))
        delay(10);
    Serial.println("HID ready");
    Serial.println("Filter initialized");
}

#define MAGIC_HEADER 0xF4
static char buf[11]; // 这也是全局的，不会在loop()中清空

void loop() {
    // 处理触摸数据 - 无延迟，实时响应
    if (Serial.available() > 0) {
        if (Serial.read() == MAGIC_HEADER) {
            // 检查是否有足够的数据可用
            if (Serial.available() >= 11) {
                Serial.readBytes(buf, 11);
                if (buf[1] == 0x03) { // 初始化屏幕的指令
                    unsigned long start = millis();
                    while (!HID.ready() && (millis() - start < 3000))
                        delay(10);
                    Serial.println("HID reinitialized with new dimensions");
                    memcpy(&report_descriptor[41], &buf[2], 4);
                    memcpy(&report_descriptor[56], &buf[5], 4);
                    Device.begin();
                    USB.begin();
                    start = millis();
                    while (!HID.ready() && (millis() - start < 3000))
                        delay(10);
                    Serial.println("HID ready");
                }
                Device.send((uint8_t *)buf, 11);
            }
        }
    }

    // 定期读取并输出手柄状态
    if (millis() - lastJoystickRead >= JOYSTICK_READ_INTERVAL) {
        outputJoystickState();
        lastJoystickRead = millis(); // 更新全局变量
    }
}