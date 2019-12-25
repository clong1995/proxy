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
	clientAddr := flag.String("client", ":80", "client address,like :80")
	serverAddr := flag.String("server", "quickex.com.cn:20080", "server address,like quickex.com.cn:20080")
	flag.Parse()
	if *clientAddr == "" || *serverAddr == "" {
		log.Fatalln("Parameters of the abnormal")
	}
	d := common.NewDelayer()
	for {
		err := dialRelay(*clientAddr, *serverAddr)
		d.ProcError(err)
	}
}

func dialRelay(clientAddr, serverAddr string) error {
	//发起连接到远程服务
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
		//设置超时
		_ = serverConn.SetReadDeadline(time.Now().Add(90 * time.Second))
		//阻塞等待读取消息，取出前四位
		_, err = io.ReadFull(serverConn, buf[:4])
		if err != nil {
			log.Println(err)
			_ = serverConn.Close()
			return err
		}
		//收到消息后，就解除了超时错误的模式，
		_ = serverConn.SetReadDeadline(time.Time{}) //解除超时，进入再次循环的阻塞模式（io.ReadFull）

		if bytes.Equal(buf[:4], []byte("PING")) { //收到ping
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

	//publicAddr 大众的ip和端口，目前没啥用，不过可以做ip端口限制

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
	go common.CopyTCPConn(clientConn, serverConn)
	//从本机返回远端
	go common.CopyTCPConn(serverConn, clientConn)
}
