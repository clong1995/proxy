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

func main() {

	publicAddr := flag.String("public", "", "public address,like 127.0.0.1:28080")
	serverAddr := flag.String("server", "", "server address,like 127.0.0.1:18080")
	flag.Parse()

	if *publicAddr == "" || *serverAddr == "" {
		log.Fatalln("Parameters of the abnormal")
	}

	/*log.SetFlags(log.Ldate | log.Llongfile)
	serverAddr, publicAddr := "192.168.20.163:46687", "192.168.20.163:8080"*/

	//对内
	serverTcpAddr, err := net.ResolveTCPAddr("tcp", *serverAddr)
	if err != nil {
		log.Fatalln(err)
	}
	serverListener, err := net.ListenTCP("tcp", serverTcpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	//对公
	publicTcpAddr, err := net.ResolveTCPAddr("tcp", *publicAddr)
	if err != nil {
		log.Fatalln(err)
	}
	publicListener, err := net.ListenTCP("tcp", publicTcpAddr)
	if err != nil {
		log.Fatalln(err)
	}

	//通道
	publicConnChan := make(chan *net.TCPConn)
	go acceptConn(publicListener, publicConnChan) //写通道

	d := common.NewDelayer()
	for {
		//阻塞对内
		serverConn, err := serverListener.AcceptTCP()
		if !d.ProcError(err) {
			go authConn(serverConn, publicConnChan) //读通道
		}
	}
}

func acceptConn(publicListener *net.TCPListener, publicConnChan chan *net.TCPConn) {
	d := common.NewDelayer()
	for {
		publicConn, err := publicListener.AcceptTCP()
		if !d.ProcError(err) {
			publicConnChan <- publicConn
		}
	}
}

func authConn(serverConn *net.TCPConn, publicConnChan chan *net.TCPConn) {
	var buf [64]byte

	for {
		select {
		//取出链接
		case publicConn := <-publicConnChan:
			publicAddr := publicConn.RemoteAddr().String()
			buf := []byte{'C', 'O', 'N', 'N',
				uint8(len(publicAddr) >> 8), uint8(len(publicAddr))}
			buf = append(buf, publicAddr...)
			_, err := serverConn.Write(buf)
			if err != nil {
				log.Println(err)
				_ = serverConn.Close()
				publicConnChan <- publicConn
				return
			}

			for {
				//阻塞等待
				_, err = io.ReadFull(serverConn, buf[:4])
				if err != nil {
					log.Println(err)
					_ = serverConn.Close()
					//重新放回到通道里
					publicConnChan <- publicConn
					return
				}

				if bytes.Equal(buf[:4], []byte("ACPT")) {
					break
				}
			}
			//从public转发到内端口
			go copyTCPConn(serverConn, publicConn)
			//等待内端口转发到public
			go copyTCPConn(publicConn, serverConn)
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
				_ = serverConn.SetReadDeadline(time.Time{})
				//回应了pang
				if bytes.Equal(buf[:4], []byte("PONG")) {
					break
				}
			}
		}
	}
}

func copyTCPConn(dst, src *net.TCPConn) {
	_, _ = io.Copy(dst, src)
	_ = src.CloseRead()
	_ = dst.CloseWrite()
}
