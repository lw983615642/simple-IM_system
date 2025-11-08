package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int
	//在线用户的列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex //go语言中所有跟同步有关的包都在sync中
	//消息广播的channel
	Message chan string
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息的goroutine，一旦有消息就发送给全部的在线user
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message
		//将msg发送给全部的在线user
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {

			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

// 写一个广播的方法
func (this *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}
func (this *Server) Handler(conn net.Conn) {
	//...当前链接的业务
	//fmt.Println("链接建立成功")
	user := NewUser(conn, this)

	user.Online()
	//监听用户是否活跃
	isLive := make(chan bool)

	//接受客户端发送的消息
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err", err)
				return
			}
			//提取用户的消息（去除/n）
			msg := string(buf[:n-1])
			//将消息广播
			user.DoMessage(msg)
			//用户任意消息证明其活跃
			isLive <- true
		}
	}()
	//当前handler阻塞
	for {
		select {
		case <-isLive:
			//说明当前用户活跃，重制定时器
			//不做任何操作，为了激活select，更新下面的定时器
		case <-time.After(time.Second * 10):
			//说明已超时
			//将当前user强制关闭
			user.SendMsg("你已被超时踢出\n")
			//销毁资源
			close(user.C)
			//关闭连接
			conn.Close()
			//退出当前handler
			return
		}
	}

}

// 启动服务器的接口
func (this *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.listen err", err)
		return
	}
	//close listen socket
	defer listener.Close()
	//启动监听message的groutine
	go this.ListenMessage()
	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}
		//do handler
		go this.Handler(conn)
	}

}
