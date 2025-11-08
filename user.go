package main

import "net"

type User struct {
	Name   string
	Addr   string
	C      chan string //表示通道里传的是string类型的消息
	conn   net.Conn
	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	//启动监听当前user channel的goroutine
	go user.ListenMessage()
	return user
}

// 用户的上线业务
func (this *User) Online() {

	//当前用户上线，将用户加入到online map中
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()
	//广播上线消息
	this.server.Broadcast(this, "已上线")
}

// 用户的下线业务
func (this *User) Offline() {

	//当前用户下线，将用户从online map中删除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()
	//广播上线消息
	this.server.Broadcast(this, "已下线")
}

// 给当前用户的客户端发送消息
func (this *User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

// 用户处理消息的业务
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onelineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线" + "\n"
			this.SendMsg(onelineMsg)
		}
		this.server.mapLock.Unlock()

	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := msg[7:]
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前name已被占用\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()
			this.Name = newName
			this.SendMsg("您已成功修改为：" + newName + "\n")
		}
	} else {
		this.server.Broadcast(this, msg)

	}
}

// 监听当前User Chanel的方法，一旦有消息，就发送给对端客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C
		this.conn.Write([]byte(msg + "\n"))

	}
}
