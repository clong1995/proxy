package main

import (
	"bytes"
	"ferry/common"
	"flag"
	"io"
	"log"
	"net"
	"time"
)

func init() {
	log.SetFlags(log.Llongfile)
}

func main() {
	//对大众的端口
	publicAddr := flag.String("public", ":10080", "public address,like :10080")
	//内应需要链接的端口
	serverAddr := flag.String("server", ":20080", "server address,like :20080")
	flag.Parse()

	if *publicAddr == "" || *serverAddr == "" {
		log.Fatalln("Parameters of the abnormal")
	}

	//创建对内应的监听
	serverTcpAddr, err := net.ResolveTCPAddr("tcp", *serverAddr)
	if err != nil {
		log.Fatalln(err)
	}
	serverListener, err := net.ListenTCP("tcp", serverTcpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	//创建对大众的监听
	publicTcpAddr, err := net.ResolveTCPAddr("tcp", *publicAddr)
	if err != nil {
		log.Fatalln(err)
	}
	publicListener, err := net.ListenTCP("tcp", publicTcpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	//给大众的保存链接的通道
	publicConnChan := make(chan *net.TCPConn)
	//开辟协程等待响应大众发送消息
	go acceptConn(publicListener, publicConnChan) //写通道

	//自增延时器
	d := common.NewDelayer()
	for {
		//监听对内服务
		serverConn, err := serverListener.AcceptTCP() //这里阻塞，等待内应激活（内应联入发送消息）
		if !d.ProcError(err) {
			//消费大众的消息
			go authConn(serverConn, publicConnChan) //读通道
		}
	}
}

//写入的消息
func acceptConn(publicListener *net.TCPListener, publicConnChan chan *net.TCPConn) {
	//自增延时器
	d := common.NewDelayer()
	for {
		//阻塞等待消息，监听对外服务
		publicConn, err := publicListener.AcceptTCP()
		if !d.ProcError(err) {
			//写入消息
			publicConnChan <- publicConn
		}
	}
}

//消费public链接的消息
func authConn(serverConn *net.TCPConn, publicConnChan chan *net.TCPConn) {
	var buf [64]byte
	for {
		select {
		//取出链接
		case publicConn := <-publicConnChan:
			//大众的ip:port
			publicAddr := publicConn.RemoteAddr().String()
			buf := []byte{'C', 'O', 'N', 'N', uint8(len(publicAddr) >> 8), uint8(len(publicAddr))}
			buf = append(buf, publicAddr...) //例如CONN09127.0.0.1
			//写到对内应
			_, err := serverConn.Write(buf)
			if err != nil {
				log.Println(err)
				_ = serverConn.Close()
				//重新放回到通道里
				publicConnChan <- publicConn
				return
			}

			for {
				//阻塞等待对内应的响应（等待内应回消息），将前4位写入buf
				_, err = io.ReadFull(serverConn, buf[:4])
				if err != nil {
					log.Println(err)
					_ = serverConn.Close()
					//重新放回到通道里
					publicConnChan <- publicConn
					return
				}
				//如果对内应的端口返回acpt，停止阻塞
				if bytes.Equal(buf[:4], []byte("ACPT")) {
					break
				}
			}
			//从public转发到内端口
			go common.CopyTCPConn(serverConn, publicConn)
			//等待内端口转发到public
			go common.CopyTCPConn(publicConn, serverConn)
			return
		//定时检测是否是在线
		case <-time.After(60 * time.Second):
			_, err := serverConn.Write([]byte("PING"))
			if err != nil {
				log.Println(err)
				_ = serverConn.Close()
				return
			}

			for {
				_ = serverConn.SetReadDeadline(time.Now().Add(90 * time.Second))
				//等待读取
				_, err = io.ReadFull(serverConn, buf[:4])
				if err != nil {
					log.Println(err)
					_ = serverConn.Close()
					return
				}
				//去掉超时
				_ = serverConn.SetReadDeadline(time.Time{})
				//回应了pang
				if bytes.Equal(buf[:4], []byte("PONG")) {
					break
				}
			}
		}
	}
}
