import { useEffect, useRef, useState } from "react";
import { Button, IconButton, Input, Paper } from "@mui/material";

const UploadButton = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: 'calc(50% - 100px)',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>上传图片</button>
}

const UploadButtonJIETU = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: '50%',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>屏幕截图</button>
}



const UploadButton5s = ({ onClick }) => {
    return <button
        style={{
            position: 'absolute',
            width: '200px',
            height: '80px',
            left: '50%',
            marginLeft: '-105px',
            top: 'calc(50% + 100px)',
            borderRadius: '50px',
            border: "5px solid #00b894",
            transition: ".25s",
            fontSize: '24px',
            background: "#2C3A47",
            color: "white",
        }}
        onClick={onClick}>5s后截图</button>
}



// const FixedIcon = (props) => {
const FixedIcon = ({ x, y, size, bgColor, textColor, text }) => {
    return <button
        style={{
            position: 'absolute',
            left: x,
            top: y,
            width: size || 28,
            height: size || 28,
            borderRadius: size || 28,
            backgroundColor: bgColor || "#d90051",
            color: textColor || "white",
            marginLeft: size / -2 || -14,
            marginTop: size / -2 || -14,
            border: "None",
            alignItems: "center",
            pointerEvents: "none",
        }}
    >
        {text}
    </button>
}
const GroupFixedIcon = ({ pos_s, bgColor, textColor, text }) => {
    return <div>
        {
            pos_s.map((pos, index) => <FixedIcon
                key={index}
                x={pos[0]}
                y={pos[1]}
                size={18}
                bgColor={bgColor}
                textColor={textColor}
                text={`${text}_${index}`}
            />)
        }
    </div>
}

const CostumedInput = ({ defaultValue, width, onCommit, all }) => {
    const [value, setValue] = useState(defaultValue)
    return <Input
        sx={{ width: width || "40px" }}
        value={value}
        onChange={(e) => {
            setValue(e.target.value)
        }}
        onFocus={(e) => {
            window.stopPreventDefault = true
        }}
        onBlur={(e) => {
            window.stopPreventDefault = false
            onCommit && onCommit(all ? value : Number(value))
        }}
        onKeyDown={(e) => {
            if (e.key === "Enter") {
                onCommit && onCommit(all ? value : Number(value))
            }
        }}
    />
}

const WheelShow = ({ x, y, range, shift_range }) => {
    const radius = range * 2
    const shift_radius = shift_range * 2
    return <div>
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: 16,
            height: 16,
            borderRadius: 16,
            marginLeft: -8,
            marginTop: -8,
            backgroundColor: "#2196F3",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: radius,
            height: radius,
            borderRadius: radius,
            marginLeft: radius / -2 - 4,
            marginTop: radius / -2 - 4,
            border: "4px solid #2196F3",
            pointerEvents: "none",
        }} />
        {
            shift_range !== 0 && <div style={{
                position: 'absolute',
                left: x,
                top: y,
                width: shift_radius,
                height: shift_radius,
                borderRadius: shift_radius,
                marginLeft: shift_radius / -2 - 4,
                marginTop: shift_radius / -2 - 4,
                border: "4px solid #512DA8",
                pointerEvents: "none",
            }} />
        }
    </div>
}


const ViewShow = ({ x, y, range, slide_range }) => {
    // 视角中心点显示
    //在屏幕上显示两条垂直线，中间是一个小的空心矩形
    return <div>
        <div style={{
            position: 'absolute',
            left: 0,
            top: y,
            width: "10000vw",
            height: 1,
            backgroundColor: "#d90051",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: 0,
            height: "10000vh",
            width: 1,
            backgroundColor: "#d90051",
            pointerEvents: "none",
        }} />
        <div style={{
            position: 'absolute',
            left: x,
            top: y,
            width: range * 2,
            height: range * 2,
            // borderRadius: range * 2,
            marginLeft: -range,
            marginTop: -range,
            border: "1px solid #d90051",
            backgroundColor: "#transparent",
            pointerEvents: "none",
        }} />
        {
            slide_range !== 0 && <div style={{
                position: 'absolute',
                left: x,
                top: y,
                width: slide_range * 2,
                height: slide_range * 2,
                marginLeft: -slide_range,
                marginTop: -slide_range,
                border: "2px solid #2196F3",
                pointerEvents: "none",
            }} />
        }
    </div>
}


export {
    UploadButton,
    UploadButtonJIETU,
    UploadButton5s,
    FixedIcon,
    GroupFixedIcon,
    CostumedInput,
    WheelShow,
    ViewShow,
}