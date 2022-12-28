package blockchain

import (
	"bytes"
	"crypto/sha256"
)

// Blockchain - Public Distributed Database
type BlockChain struct {
	Blocks []*Block
}

type Block struct {
	Hash     []byte // Hash of the Block
	Data     []byte // Contents of the Block
	PrevHash []byte // Last Block Hash -> Link the Blocks
	Nonce    int
}

// Caculate the Hash Value of the Block
func (b *Block) DeriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

// Create Block- Params(data,prevHash)
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

func (chain *BlockChain) AddBlock(data string) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, new)
}

func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}
