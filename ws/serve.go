package ws

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-gin-chat/models"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// 客户端连接详情
type wsClients struct {
	conn *websocket.Conn

	RemoteAddr string

	uid float64

	username string

	roomId string
}

// 存放客户端连接
type ws struct {
	clients []wsClients
}

// client & serve 的消息体
type msg struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
}

// 变量定义初始化
var (
	wsc = ws{}

	wsUpgrader = websocket.Upgrader{}

	clientMsg = msg{}

	serveMsg = msg{}

	mutex  = sync.Mutex{}

	rooms = [100][]wsClients{}
)

// 定义消息类型
const msgTypeOnline = 1  // 上线
const msgTypeOffline = 2 // 离线
const msgTypeSend = 3    // 消息发送

func Run(gin *gin.Context) {

	// @see https://github.com/gorilla/websocket/issues/523
	wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }

	c, _ := wsUpgrader.Upgrade(gin.Writer, gin.Request, nil)

	defer c.Close()

	mainProcess(c)
}

// 主程序，负责循环读取客户端消息跟消息的发送
func mainProcess(c *websocket.Conn) {
	for {
		_, message, err := c.ReadMessage()
		serveMsgStr := message

		// 处理心跳响应 , heartbeat为与客户端约定的值
		if string(serveMsgStr) == `heartbeat` {
			c.WriteMessage(websocket.TextMessage, []byte(`{"status":0,"data":"heartbeat ok"}`))
			continue
		}

		json.Unmarshal(message, &clientMsg)
		if clientMsg.Data == nil {
			return
			//mainProcess(c)
		}
		//log.Println("来自客户端的消息", clientMsg)

		if err != nil { // 离线通知
			log.Println("ReadMessage error1", err)
			disconnect(c)
			c.Close()
			return
		}

		if clientMsg.Status == msgTypeOnline { // 进入房间，建立连接
			handleConnClients(c)
			serveMsgStr = formatServeMsgStr(msgTypeOnline)
		}

		if clientMsg.Status == msgTypeSend { // 消息发送
			serveMsgStr = formatServeMsgStr(msgTypeSend)
		}

		//log.Println("serveMsgStr", string(serveMsgStr))
		notify(c, string(serveMsgStr))
	}
}



// 处理建立连接的用户
func handleConnClients(c *websocket.Conn) {
	roomId, roomIdInt := getRoomId()

	for cKey, wcl := range rooms[roomIdInt] {
		if wcl.uid == clientMsg.Data.(map[string]interface{})["uid"].(float64) {
			mutex.Lock()
			// 通知当前用户下线
			wcl.conn.WriteMessage(websocket.TextMessage, []byte(`{"status":-1,"data":[]}`))
			rooms[roomIdInt] = append(rooms[roomIdInt][:cKey], rooms[roomIdInt][cKey+1:]...)
			mutex.Unlock()
		}
	}

	mutex.Lock()
	rooms[roomIdInt] = append(rooms[roomIdInt], wsClients{
		conn:       c,
		RemoteAddr: c.RemoteAddr().String(),
		uid:        clientMsg.Data.(map[string]interface{})["uid"].(float64),
		username:   clientMsg.Data.(map[string]interface{})["username"].(string),
		roomId:     roomId,
	})
	mutex.Unlock()
}

// 统一消息发放
func notify(conn *websocket.Conn, msg string) {
	_, roomIdInt := getRoomId()
	for _, con := range rooms[roomIdInt] {
		if con.RemoteAddr != conn.RemoteAddr().String() {
			con.conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}
}

// 离线通知
func disconnect(conn *websocket.Conn) {
	_, roomIdInt := getRoomId()
	for index, con := range rooms[roomIdInt] {
		if con.RemoteAddr == conn.RemoteAddr().String() {
			data := map[string]interface{}{
				"username": con.username,
				"uid":      con.uid,
				"time":     time.Now().UnixNano() / 1e6, // 13位  10位 => now.Unix()
			}

			jsonStrServeMsg := msg{
				Status: msgTypeOffline,
				Data:   data,
			}
			serveMsgStr, _ := json.Marshal(jsonStrServeMsg)

			disMsg := string(serveMsgStr)

			mutex.Lock()
			rooms[roomIdInt] = append(rooms[roomIdInt][:index], rooms[roomIdInt][index+1:]...)
			mutex.Unlock()
			notify(conn, disMsg)
		}
	}
}


// 格式化传送给客户端的消息数据
func formatServeMsgStr(status int) []byte {

	data := map[string]interface{}{
		"username": clientMsg.Data.(map[string]interface{})["username"].(string),
		"uid":      clientMsg.Data.(map[string]interface{})["uid"].(float64),
		"room_id":      clientMsg.Data.(map[string]interface{})["room_id"].(string),
		"time":     time.Now().UnixNano() / 1e6, // 13位  10位 => now.Unix()
	}

	if status == msgTypeSend {
		data["avatar_id"] = clientMsg.Data.(map[string]interface{})["avatar_id"].(string)
		data["content"] = clientMsg.Data.(map[string]interface{})["content"].(string)

		// 保存消息
		stringUid := strconv.FormatFloat(data["uid"].(float64), 'f', -1, 64)
		intUid, _ := strconv.Atoi(stringUid)


		if _, ok := clientMsg.Data.(map[string]interface{})["image_url"]; ok {
			// 存在图片
			models.SaveContent(map[string]interface{}{
				"user_id": intUid,
				"content": data["content"],
				"room_id": data["room_id"],
				"image_url": clientMsg.Data.(map[string]interface{})["image_url"].(string),
			})
		}else{
			models.SaveContent(map[string]interface{}{
				"user_id": intUid,
				"room_id": data["room_id"],
				"content": data["content"],
			})
		}

	}

	jsonStrServeMsg := msg{
		Status: status,
		Data:   data,
	}
	serveMsgStr, _ := json.Marshal(jsonStrServeMsg)

	return serveMsgStr
}

func getRoomId() (string, int) {
	roomId := clientMsg.Data.(map[string]interface{})["room_id"].(string)

	roomIdInt, _ := strconv.Atoi(roomId)
	return roomId, roomIdInt
}

// 对外方法
//  获取在线用户列表，暂不实现
func OnLineUserList() {
	var uids = []float64{}
	for _, cl := range wsc.clients {
		uids = append(uids, cl.uid)
	}
	models.GetOnlineUserList(uids)
}

func GetOnlineUserCount() int {
	return len(wsc.clients)
}

func GetOnlineRoomUserCount(roomId int) int {
	return len(rooms[roomId])
}