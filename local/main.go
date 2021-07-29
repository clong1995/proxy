package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
	"proxy/common"
	"time"
)

func init() {
	log.SetFlags(log.Llongfile)
}

func main() {
	//连接到本地端口
	localAddr := flag.String("local", ":80", "local address,like 127.0.0.1:8080")
	//连接到公网服务器
	serverAddr := flag.String("server", "1.2.3.4:201", "server address,like 1.2.3.4:201")
	flag.Parse()
	if *localAddr == "" || *serverAddr == "" {
		log.Fatalln("Parameters of the abnormal")
	}
	d := common.NewDelayer()
	for {
		err := dialRelay(*localAddr, *serverAddr)
		d.ProcError(err)
	}
}

func dialRelay(localAddr, serverAddr string) (err error) {
	//发起连接到远程服务
	serverTCPAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		log.Println(err)
		return
	}
	serverTCPConn, err := net.DialTCP("tcp", nil, serverTCPAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err != nil {
			err = serverTCPConn.Close()
			log.Println(err)
		}
	}()

	var buf [4]byte

	for {
		//设置超时
		if err = serverTCPConn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
			log.Println(err)
			return
		}

		//阻塞等待读取消息，取出前四位
		if _, err = io.ReadFull(serverTCPConn, buf[:4]); err != nil {
			log.Println(err)
			return
		}

		//收到消息后，就解除了超时错误的模式，
		if err = serverTCPConn.SetReadDeadline(time.Time{}); err != nil { //解除超时，进入再次循环的阻塞模式(io.ReadFull)
			log.Println(err)
			return
		}

		if bytes.Equal(buf[:4], common.PING) { //收到ping
			if _, err = serverTCPConn.Write(common.PONG); err != nil { //发送pang
				log.Println(err)
				return
			}
		} else if bytes.Equal(buf[:4], common.CONN) { //收到conn
			go acceptConn(serverTCPConn, localAddr)
			return
		}
	}
}

func acceptConn(serverTCPConn *net.TCPConn, localAddr string) {
	var err error
	defer func() {
		if err != nil {
			err = serverTCPConn.Close()
			log.Println(err)
		}
	}()
	var addrLen [2]byte
	if _, err = io.ReadFull(serverTCPConn, addrLen[:2]); err != nil {
		log.Println(err)
		return
	}

	//publicAddr 大众的ip和端口，目前没啥用，不过可以做ip端口限制
	publicAddr := make([]byte, (int(addrLen[0])<<8)|int(addrLen[1]))
	if _, err = io.ReadFull(serverTCPConn, publicAddr); err != nil {
		log.Println(err)
		return
	}

	if _, err = serverTCPConn.Write(common.ACPT); err != nil {
		log.Println(err)
		return
	}

	//链接本机
	localTCPAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		log.Println(err)
		return
	}
	localTCPConn, err := net.DialTCP("tcp", nil, localTCPAddr)
	if err != nil {
		log.Println(err)
		return
	}

	//从远端发到本地
	go common.CopyTCPConn(localTCPConn, serverTCPConn)
	//从本机返回远端
	go common.CopyTCPConn(serverTCPConn, localTCPConn)
}
