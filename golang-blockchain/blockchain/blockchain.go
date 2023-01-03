package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

// Blockchain - Public Distributed Database
type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

// Iterate through Blockchain using currentHash -> prevHash
type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// if DB exists, return true
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// Bring the exisiting Blockchain
func ContinueBlockChain(address string) *BlockChain {
	if DBexists() == false {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	var lastHash []byte
	// open DB by path
	opts := badger.DefaultOptions(dbPath)

	db, err := badger.Open(opts)
	Handle(err)

	// get the lastHash
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	Handle(err)

	chain := BlockChain{lastHash, db}

	return &chain
}

// Add Block to the Blockchain
func (chain *BlockChain) AddBlock(transaction []*Transaction) {
	var lastHash []byte

	// using "lh" value, bring the lastHash
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.ValueCopy(nil)

		return err
	})
	Handle(err)

	// Create a Block with data, LastHash
	newBlock := CreateBlock(transaction, lastHash)

	// Set blockchain LastHash to New Block's Hash value
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)

}
func InitBlockChain(address string) *BlockChain {
	// blockchain using badger DB
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	// Open the Database
	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handle(err)

	// Check if the blockchain already exist
	// if exists, get the lastHash of the last block in the blockchain
	// if not, create a genesis block and create a new blockchain and returns it
	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})
	Handle(err)

	blockchain := BlockChain{lastHash, db}

	return &blockchain
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		encodedBlock, err := item.ValueCopy(nil)
		block = Deserialize(encodedBlock)

		return err
	})
	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}

func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction

	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		// 블록을 순회하면서 탐색
		for _, tx := range block.Transaction {
			txID := hex.EncodeToString(tx.ID)
		// 블록 내의 거래들을 살펴본다
		Outputs:
			// 블록 내 거래의 출력들을 살펴본다
			for outIdx, out := range tx.Outputs {
				// 해당 거래가 사용하지 않은 출력을 포함하고 있는지를 확인한다
				if spentTXOs[txID] != nil {
					// 만약에 해당 거래내 에서 사용한 출력과 현재 출력이 동일하면 쓰지 못하므로 
					// 반복문을 진행한다
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				// 해당 출력이 우리의 키로 잠궈저 있는지 확인
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			// 입력이 있으면 출력이 무조건 있기 때문에
			// 우리가 해당 입력을 풀 수 있다 -> 해당 입력과 연결된 출력은 이미 사용한 것이므로 체크해준다
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}

		// 끝까지 다 돌면 나가기
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxs
}

// 해당 함수는 체인에서 사용하지 않은 출력들만 찾는 기능을 한다
func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	// 해당 함수를 호출하여 사용하지 않은 출력을 가지고 있는 거래를 가져온다
	unspentTransactions := chain.FindUnspentTransactions(address)

	// 해당 거래를 순회하면서 해당 거래의 출력이 우리가 가지고 있는 키로 풀 수 있으면 사용가능한 출력
	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}
