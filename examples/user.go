package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	conn, err := net.Dial("tcp", "127.0.0.1:4785")
	if err != nil {
		panic(err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println("read", buf[:n])
		}
	}()

	for {
		time.Sleep(time.Millisecond * time.Duration(rand.Int()%500+500))
		_, err := conn.Write([]byte{2, 3, 4, 5, 6})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
