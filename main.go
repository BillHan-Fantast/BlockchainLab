package main

import (
	"flag"
	"fmt"
	"lab2.com/blockchain/chain"
	"log"
	"net/http"
	"strings"
)

func main() {
	serverPort := flag.String("port", "8000", "http port number where server will run")
	flag.Parse()

	blockchain := chain.InitBlockChain()
	nodeID := strings.Replace(chain.PseudoUUID(), "-", "", -1)

	log.Printf("Starting gochain HTTP Server. Listening at port %q", *serverPort)

	http.Handle("/", chain.NewHandler(&blockchain, nodeID))
	http.ListenAndServe(fmt.Sprintf(":%s", *serverPort), nil)
}
