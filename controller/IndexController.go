package controller

import (
	"github.com/gin-gonic/gin"
	"go-gin-chat/services/message_service"
	"go-gin-chat/services/user_service"
	"go-gin-chat/ws"
	"net/http"
)

func Index(c *gin.Context) {
    // 已登录跳转room界面，多页面应该考虑放在中间件实现
	userInfo := user_service.GetUserInfo(c)
	if len(userInfo) > 0  {
		c.Redirect(http.StatusFound,"/home")
		return
	}

	OnlineUserCount := ws.GetOnlineUserCount()

	c.HTML(http.StatusOK, "login.html", gin.H{
		"OnlineUserCount": OnlineUserCount,
	})
}

func Login(c *gin.Context) {
	user_service.Login(c)
}

func Logout(c *gin.Context) {
	user_service.Logout(c)
}

func Home(c *gin.Context) {
	userInfo := user_service.GetUserInfo(c)
	rooms := []map[string]interface{}{
		{"id": 1, "num": ws.GetOnlineRoomUserCount(1)},
		{"id": 2, "num": ws.GetOnlineRoomUserCount(2)},
		{"id": 3, "num": ws.GetOnlineRoomUserCount(3)},
		{"id": 4, "num": ws.GetOnlineRoomUserCount(4)},
		{"id": 5, "num": ws.GetOnlineRoomUserCount(5)},
		{"id": 6, "num": ws.GetOnlineRoomUserCount(6)},
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"rooms": rooms,
		"user_info": userInfo,
	})
}

func Room(c *gin.Context) {
	roomId := c.Param("room_id")

	userInfo := user_service.GetUserInfo(c)
	msgList := message_service.GetLimitMsg(roomId)

	c.HTML(http.StatusOK, "room.html", gin.H{
		"user_info": userInfo,
		"msg_list":msgList,
		"room_id":roomId,
	})
}
