package main

import "github.com/yddeng/gateway"

func main() {
	gateway.Gate("127.0.0.1:4784", "127.0.0.1:4785")
	select {}
}
