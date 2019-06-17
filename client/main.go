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

	clientAddr := flag.String("client", "", "client address,like 127.0.0.1:8080")
	serverAddr := flag.String("server", "", "server address,like 123.123.123.123:18080")
	flag.Parse()

	if *clientAddr == "" || *serverAddr == "" {
		log.Fatalln("Parameters of the abnormal")
	}

	/*log.SetFlags(log.Ldate | log.Llongfile)
	clientAddr, serverAddr := "127.0.0.1:8009", "192.168.20.163:46687"*/

	d := common.NewDelayer()
	for {
		err := dialRelay(*clientAddr, *serverAddr)
		d.ProcError(err)
	}
}

func dialRelay(clientAddr, serverAddr string) error {
	//发起连接
	serverTcpAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		log.Println(err)
		return err
	}
	serverConn, err := net.DialTCP("tcp", nil, serverTcpAddr)
	if err != nil {
		log.Println(err)
		return err
	}

	var buf [64]byte

	for {
		_ = serverConn.SetReadDeadline(time.Now().Add(90 * time.Second))
		//收到消息后，就解除了超时错误的模式
		_, err = io.ReadFull(serverConn, buf[:4])
		if err != nil {
			log.Println(err)
			_ = serverConn.Close()
			return err
		}
		_ = serverConn.SetReadDeadline(time.Time{}) //恢复为阻塞模式。

		//log.Println("===>",string(buf[:4]))

		if bytes.Equal(buf[:4], []byte("PING")) { //收到平
			_, err = serverConn.Write([]byte("PONG")) //发送pang
			if err != nil {
				log.Println(err)
				_ = serverConn.Close()
				return err
			}
		} else if bytes.Equal(buf[:4], []byte("CONN")) { //收到conn
			go acceptConn(serverConn, clientAddr)
			return nil
		}
	}
}

func acceptConn(serverConn *net.TCPConn, clientAddr string) {
	var addrLen [2]byte
	_, err := io.ReadFull(serverConn, addrLen[:2])
	if err != nil {
		_ = serverConn.Close()
		log.Println(err)
		return
	}

	publicAddr := make([]byte, (int(addrLen[0])<<8)|int(addrLen[1]))
	_, err = io.ReadFull(serverConn, publicAddr)
	if err != nil {
		_ = serverConn.Close()
		log.Println(err)
		return
	}

	_, err = serverConn.Write([]byte("ACPT"))
	if err != nil {
		_ = serverConn.Close()
		log.Println(err)
		return
	}

	//链接本机
	clientTcpAddr, err := net.ResolveTCPAddr("tcp", clientAddr)
	if err != nil {
		_ = serverConn.Close()
		log.Println(err)
		return
	}
	clientConn, err := net.DialTCP("tcp", nil, clientTcpAddr)
	if err != nil {
		_ = serverConn.Close()
		log.Println(err)
		return
	}

	//从远端发到本地
	go copyTCPConn(clientConn, serverConn)
	//从本机返回远端
	go copyTCPConn(serverConn, clientConn)
}

func copyTCPConn(dst, src *net.TCPConn) {
	_, _ = io.Copy(dst, src)
	_ = src.CloseRead()
	_ = dst.CloseWrite()
}
