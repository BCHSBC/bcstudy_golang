package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

// 내가 1 코인이 있는 상태에서 0.7코인을 보낼려고 하면
// 1코인 인풋과 0.7, 0.3 아웃풋이 나온다
// 0.7은 상대방에게 보낼 거래고, 0.3은 다시 나에게 보내는 거래이다
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// Pubkey는 상대방의 공개키라고 생각하자(암호로 값을 잠근다)
// Value는 말 그대로 얼마를 보낼지를 작성
type TxOutput struct {
	Value  int
	PubKey string
}

// 모든 거래의 입력은 이전 거래의 출력으로 구성이된다
// ID는 거래를 구별하기 위한 것이고
// OUT은 이전 거래의 출력을 가리키는 인덱스를 저장하고 있다
type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

// 거래의 모든 내용을 해시화해서 ID로 저장
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// 최초의 출력을 생성하는 거래
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	// 채굴한 사람에게 100의 보상을 준다
	txin := TxInput{[]byte{}, -1, data}
	txout := TxOutput{100, to}

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{txout}}
	tx.SetID()

	return &tx
}

// 최초의 거래인지를 확인
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}

// 새로운 거래를 생성하는 메소드
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	// 사용 가능한 출력들을 찾는다
	acc, validOutputs := chain.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("Error: not Enough money")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)

		// 새로운 입력들을 만든다
		for _, out := range outs {
			input := TxInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	// 새로운 출력을 만든다
	outputs = append(outputs, TxOutput{amount, to})

	// 내가 사용가능한것들을 모은게 보내는 것보다 많으면
	// 즉 거스름돈이 생기면
	// 나에게 다시 보내는 거래를 만든다
	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx
}
