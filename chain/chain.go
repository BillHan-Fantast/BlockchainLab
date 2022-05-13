package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/url"
)

const Difficulty = 15

const (
	NO_ERROR = iota
	ERR_WRONG_HASH
	ERR_PREFIX_ZERO_SHORT
	ERR_PREV_HASH_MISMATCH
)

type Block struct {
	Hash, Data, PrevHash []byte
	Nonce                int
}

type BlockChain struct {
	Blocks []Block
	nodes  StringSet
}

func ToHex(num int) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, int32(num))
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func (b *Block) DeriveHash() []byte {
	info := bytes.Join(
		[][]byte{
			b.PrevHash,
			b.Data,
			ToHex(b.Nonce),
			ToHex(Difficulty),
		},
		[]byte{},
	)
	hash := sha256.Sum256(info)
	return hash[:]
}

func CreateBlock(data string, prevHash []byte, nonce int) Block {
	block := Block{[]byte{}, []byte(data), prevHash, nonce}
	block.Hash = block.DeriveHash()
	return block
}

func (b *Block) Validate(checkHashCorrect bool) int {
	var intHash big.Int
	hash := b.DeriveHash()
	if checkHashCorrect && bytes.Compare(hash, b.Hash) != 0 {
		return ERR_WRONG_HASH // Hash calculation wrong
	}
	intHash.SetBytes(hash)
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))
	if intHash.Cmp(target) == -1 {
		return NO_ERROR
	} else {
		return ERR_PREFIX_ZERO_SHORT // Prefix zero not enough
	}
}

func InitBlockChain() BlockChain {
	return BlockChain{[]Block{CreateBlock("Genesis", []byte{}, 0)}, NewStringSet()}
}

func (c *BlockChain) PrevHash() []byte {
	return c.Blocks[len(c.Blocks)-1].Hash
}

func (c *BlockChain) LastBlock() Block {
	return c.Blocks[len(c.Blocks)-1]
}

func (c *BlockChain) AddBlock(b Block) int {
	if bytes.Compare(c.PrevHash(), b.PrevHash) != 0 {
		return ERR_PREV_HASH_MISMATCH // prev_block.Hash != block.PrevHash
	}
	res := b.Validate(true)
	if res != NO_ERROR {
		return res
	}
	c.Blocks = append(c.Blocks, b)
	return NO_ERROR
}

func (c *BlockChain) ValidChain(chain *[]Block) bool {
	lastBlock := (*chain)[0]
	currentIndex := 1
	for currentIndex < len(*chain) {
		block := (*chain)[currentIndex]
		// Check that the hash of the block is correct
		if bytes.Compare(block.PrevHash, lastBlock.DeriveHash()) != 0 {
			return false
		}
		// Check that the Proof of Work is correct
		if block.Validate(true) != NO_ERROR {
			return false
		}
		lastBlock = block
		currentIndex += 1
	}
	return true
}

func (bc *BlockChain) RegisterNode(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	return bc.nodes.Add(u.Host)
}

func (bc *BlockChain) ResolveConflicts() bool {
	neighbours := bc.nodes
	newChain := make([]Block, 0)

	// We're only looking for chains longer than ours
	maxLength := len(bc.Blocks)

	// Grab and verify the chains from all the nodes in our network
	for _, node := range neighbours.Keys() {
		otherBlockchain, err := findExternalChain(node)
		if err != nil {
			continue
		}

		// Check if the length is longer and the chain is valid
		if otherBlockchain.Length > maxLength && bc.ValidChain(&otherBlockchain.Chain) {
			maxLength = otherBlockchain.Length
			newChain = otherBlockchain.Chain
		}
	}
	// Replace our chain if we discovered a new, valid chain longer than ours
	if len(newChain) > 0 {
		bc.Blocks = newChain
		return true
	}

	return false
}

type blockchainInfo struct {
	Length int     `json:"length"`
	Chain  []Block `json:"chain"`
}

func findExternalChain(address string) (blockchainInfo, error) {
	response, err := http.Get(fmt.Sprintf("http://%s/chain", address))
	if err == nil && response.StatusCode == http.StatusOK {
		var bi blockchainInfo
		if err := json.NewDecoder(response.Body).Decode(&bi); err != nil {
			return blockchainInfo{}, err
		}
		return bi, nil
	}
	return blockchainInfo{}, err
}

func FindNonce(b Block) int {
	nonce := 0
	for nonce < math.MaxInt64 {
		b.Nonce = nonce
		res := b.Validate(false)
		if res == NO_ERROR {
			break
		}
		nonce++
	}
	return nonce
}
