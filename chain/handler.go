package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func NewHandler(blockchain *BlockChain, nodeID string) http.Handler {
	h := handler{blockchain, nodeID}

	mux := http.NewServeMux()
	mux.HandleFunc("/nodes/register", buildResponse(h.RegisterNode))
	mux.HandleFunc("/nodes/resolve", buildResponse(h.ResolveConflicts))
	mux.HandleFunc("/mine", buildResponse(h.Mine))
	mux.HandleFunc("/chain", buildResponse(h.Blockchain))
	return mux
}

type handler struct {
	blockchain *BlockChain
	nodeId     string
}

type response struct {
	value      interface{}
	statusCode int
	err        error
}

func buildResponse(h func(io.Writer, *http.Request) response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := h(w, r)
		msg := resp.value
		if resp.err != nil {
			msg = resp.err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.statusCode)
		if err := json.NewEncoder(w).Encode(msg); err != nil {
			log.Printf("could not encode response to output: %v", err)
		}
	}
}

func (h *handler) Mine(w io.Writer, r *http.Request) response {
	if r.Method != http.MethodGet {
		return response{
			nil,
			http.StatusMethodNotAllowed,
			fmt.Errorf("method %s not allowd", r.Method),
		}
	}

	init_time := time.Now()

	// We run the proof of work algorithm to get the next nonce...
	block := CreateBlock("Mined_Block", h.blockchain.PrevHash(), 0)
	nonce := FindNonce(block)
	block.Nonce = nonce
	block.Hash = block.DeriveHash()

	// Forge the new Block by adding it to the chain
	res := h.blockchain.AddBlock(block)
	ellaps := time.Since(init_time)

	log.Println("Mining some coins. Using time: ", ellaps)
	if res != NO_ERROR {
		resp := map[string]interface{}{"message": "New Block Forged", "block": block}
		return response{resp, http.StatusOK, fmt.Errorf("Block not added")}
	}
	resp := map[string]interface{}{"message": "New Block Forged", "block": block}
	return response{resp, http.StatusOK, nil}
}

func (h *handler) Blockchain(w io.Writer, r *http.Request) response {
	if r.Method != http.MethodGet {
		return response{
			nil,
			http.StatusMethodNotAllowed,
			fmt.Errorf("method %s not allowd", r.Method),
		}
	}
	log.Println("Blockchain requested")

	resp := map[string]interface{}{"chain": h.blockchain.Blocks, "length": len(h.blockchain.Blocks)}
	return response{resp, http.StatusOK, nil}
}

func (h *handler) RegisterNode(w io.Writer, r *http.Request) response {
	if r.Method != http.MethodPost {
		return response{
			nil,
			http.StatusMethodNotAllowed,
			fmt.Errorf("method %s not allowd", r.Method),
		}
	}

	log.Println("Adding node to the blockchain")

	var body map[string]string
	err := json.NewDecoder(r.Body).Decode(&body)

	h.blockchain.RegisterNode(body["node"])

	resp := map[string]interface{}{
		"message": "New nodes have been added",
		"nodes":   h.blockchain.nodes.Keys(),
	}

	status := http.StatusCreated
	if err != nil {
		status = http.StatusInternalServerError
		err = fmt.Errorf("fail to register nodes")
		log.Printf("there was an error when trying to register a new node %v\n", err)
	}

	return response{resp, status, err}
}

func (h *handler) ResolveConflicts(w io.Writer, r *http.Request) response {
	if r.Method != http.MethodGet {
		return response{
			nil,
			http.StatusMethodNotAllowed,
			fmt.Errorf("method %s not allowd", r.Method),
		}
	}

	log.Println("Resolving blockchain differences by consensus")

	msg := "Our chain is authoritative"
	if h.blockchain.ResolveConflicts() {
		msg = "Our chain was replaced"
	}

	resp := map[string]interface{}{"message": msg, "chain": h.blockchain.Blocks}
	return response{resp, http.StatusOK, nil}
}
