package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pelletier/go-toml/v2"
)

// Config 对应 settings.toml 的结构
type Config struct {
	Listen     string `toml:"listen" comment:"WS 代理监听地址 (接收来自 CDN 的连接)"`
	Path       string `toml:"path" comment:"WS 路由路径"`
	Backend    string `toml:"backend" comment:"MC 服务器真实的内网 TCP 地址"`
	Header     string `toml:"header" comment:"CDN 传递真实 IP 的 Header 名称 (例如 X-Forwarded-For 或 CF-Connecting-IP)"`
	PortHeader string `toml:"port_header" comment:"CDN 传递真实端口的 Header 名称 (留空则默认使用临时端口 50000)"`
	Password   string `toml:"password" comment:"安全校验密钥：CDN 请求必须携带 X-Toh-Token 且值匹配才允许连接 (留空则不校验)"`
}

var (
	globalConfig Config
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // 允许跨域（CDN 转发过来）
	}
)

const configFile = "settings.toml"

func main() {
	// 1. 检查并生成默认配置文件
	ensureConfigFile()

	// 2. 加载 TOML 配置文件
	fileBytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("[FATAL] 读取配置文件失败: %v", err)
	}
	if err := toml.Unmarshal(fileBytes, &globalConfig); err != nil {
		log.Fatalf("[FATAL] 解析配置文件失败: %v", err)
	}

	// 4. 启动服务
	http.HandleFunc(globalConfig.Path, handleWebSocket)

	log.Printf("[PROXY] 正在监听 WS: http://%s%s", globalConfig.Listen, globalConfig.Path)
	log.Printf("[PROXY] 后端 MC 服务端目标: %s", globalConfig.Backend)
	if globalConfig.Password != "" {
		log.Printf("[SECURITY] 已启用 X-Toh-Token 密码校验机制")
	}

	if err := http.ListenAndServe(globalConfig.Listen, nil); err != nil {
		log.Fatalf("[FATAL] 监听失败: %v", err)
	}
}

// ensureConfigFile 检查配置文件是否存在，若不存在则使用 v2 encoder 自动生成
func ensureConfigFile() {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("[INFO] 未检测到配置文件，正在生成默认的 %s ...", configFile)

		// 定义默认结构体数据
		defaultConfig := Config{
			Listen:     ":8080",
			Path:       "/mc",
			Backend:    "127.0.0.1:25566",
			Header:     "X-Forwarded-For",
			PortHeader: "X-Forwarded-Port",
			Password:   "",
		}

		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)

		if err := encoder.Encode(defaultConfig); err != nil {
			log.Fatalf("[FATAL] 序列化默认配置失败: %v", err)
		}

		if err := os.WriteFile(configFile, buf.Bytes(), 0644); err != nil {
			log.Fatalf("[FATAL] 自动创建配置文件失败: %v", err)
		}

		log.Printf("[INFO] 默认配置文件生成成功，程序将使用默认参数继续启动。")
	}
}

// 处理来自 CDN 的 WebSocket 请求
func mainHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 安全校验：检查密码是否匹配
	if globalConfig.Password != "" {
		reqPassword := r.Header.Get("X-Toh-Token")
		if reqPassword != globalConfig.Password {
			log.Printf("[UNAUTHORIZED] 拒绝了来自 %s 的连接请求: X-Toh-Token 校验失败", r.RemoteAddr)
			http.Error(w, "401 Unauthorized: Invalid proxy password", http.StatusUnauthorized)
			return
		}
	}

	// 2. 升级为 WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERR] WS 升级失败: %v", err)
		return
	}
	defer wsConn.Close()

	clientIP, clientPort := parseRealClientAddr(r)

	// 3. 连接到后端的 MC 服务器
	mcConn, err := net.Dial("tcp", globalConfig.Backend)
	if err != nil {
		log.Printf("[ERR] 无法连接到后端的 MC 服务器 (%s): %v", globalConfig.Backend, err)
		return
	}
	defer mcConn.Close()

	// 4. 构建并写入 PROXY Protocol v2 二进制头部给 MC 服务器
	pp2Header := buildProxyProtocolV2(clientIP, clientPort, mcConn.RemoteAddr().String())
	if _, err := mcConn.Write(pp2Header); err != nil {
		log.Printf("[ERR] 写入 PROXY v2 头部失败: %v", err)
		return
	}

	pipe(wsConn, mcConn)
}

// 兼容不同的 CDN Header 提取玩家真实 IP 与端口
func parseRealClientAddr(r *http.Request) (string, uint16) {
	ipStr, portStr, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ipStr = r.RemoteAddr
		portStr = "50000"
	}

	if h := r.Header.Get(globalConfig.Header); h != "" {
		parts := strings.Split(h, ",")
		ipStr = strings.TrimSpace(parts[0])
	}

	if globalConfig.PortHeader != "" {
		if p := r.Header.Get(globalConfig.PortHeader); p != "" {
			portStr = p
		}
	}

	port, _ := strconv.Atoi(portStr)
	if port <= 0 || port > 65535 {
		port = 50000
	}

	return ipStr, uint16(port)
}

// 严格按照 HAProxy 规范构建 PROXY protocol v2 二进制数据包
func buildProxyProtocolV2(srcIPStr string, srcPort uint16, dstAddrStr string) []byte {
	signature := []byte("\x0D\x0A\x0D\x0A\x00\x0D\x0A\x51\x55\x49\x54\x0A")

	srcIP := net.ParseIP(srcIPStr)
	dstIPStr, dstPortStr, _ := net.SplitHostPort(dstAddrStr)
	dstIP := net.ParseIP(dstIPStr)
	dstPort, _ := strconv.Atoi(dstPortStr)

	var verCmd byte = 0x21
	var famProto byte = 0x11
	var addressLen uint16 = 12

	if srcIP.To4() == nil && srcIP.To16() != nil {
		famProto = 0x21
		addressLen = 36
	}

	headerLen := 12 + 1 + 1 + 2 + int(addressLen)
	buf := make([]byte, headerLen)

	copy(buf[0:12], signature)
	buf[12] = verCmd
	buf[13] = famProto
	binary.BigEndian.PutUint16(buf[14:16], addressLen)

	pos := 16
	if famProto == 0x11 {
		copy(buf[pos:pos+4], srcIP.To4())
		pos += 4
		copy(buf[pos:pos+4], dstIP.To4())
		pos += 4
	} else {
		copy(buf[pos:pos+16], srcIP.To16())
		pos += 16
		copy(buf[pos:pos+16], dstIP.To16())
		pos += 16
	}

	binary.BigEndian.PutUint16(buf[pos:pos+2], srcPort)
	pos += 2
	binary.BigEndian.PutUint16(buf[pos:pos+2], uint16(dstPort))

	return buf
}

// 桥接 WebSocket 和 TCP Socket 的流量
func pipe(wsConn *websocket.Conn, tcpConn net.Conn) {
	errChan := make(chan error, 2)

	go func() {
		for {
			msgType, payload, err := wsConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
				_, err = tcpConn.Write(payload)
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := tcpConn.Read(buf)
			if n > 0 {
				err = wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
				if err != nil {
					errChan <- err
					return
				}
			}
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	<-errChan
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	mainHandler(w, r)
}
