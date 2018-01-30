package runtime_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/contracts"
	"github.com/vechain/thor/genesis"
	"github.com/vechain/thor/lvldb"
	"github.com/vechain/thor/runtime"
	"github.com/vechain/thor/state"
	"github.com/vechain/thor/thor"
	"github.com/vechain/thor/tx"
)

func TestCall(t *testing.T) {
	kv, _ := lvldb.NewMem()

	b0, err := genesis.Mainnet.Build(state.NewCreator(kv))
	if err != nil {
		t.Fatal(err)
	}

	state, _ := state.New(b0.Header().StateRoot(), kv)

	rt := runtime.New(state,
		thor.Address{}, 0, 0, 0, func(uint32) thor.Hash { return thor.Hash{} })

	addr := thor.BytesToAddress([]byte("acc1"))
	amount := big.NewInt(1000 * 1000 * 1000 * 1000)

	{
		// charge
		out := rt.Call(
			contracts.Energy.PackCharge(
				addr,
				amount,
			),
			0, 1000000, contracts.Energy.Address, new(big.Int), thor.Hash{})
		if out.VMErr != nil {
			t.Fatal(out.VMErr)
		}
	}
	{

		out := rt.StaticCall(
			contracts.Energy.PackBalanceOf(addr),
			0, 1000000, thor.Address{}, new(big.Int), thor.Hash{})
		if out.VMErr != nil {
			t.Fatal(out.VMErr)
		}

		var retAmount *big.Int
		if err := contracts.Energy.ABI.Unpack(&retAmount, "balanceOf", out.Value); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, amount, retAmount)
	}
}

func TestExecuteTransaction(t *testing.T) {

	kv, _ := lvldb.NewMem()

	key, _ := crypto.GenerateKey()
	addr1 := thor.Address(crypto.PubkeyToAddress(key.PublicKey))
	addr2 := thor.BytesToAddress([]byte("acc2"))
	balance1 := big.NewInt(1000 * 1000 * 1000)

	b0, err := new(genesis.Builder).
		Alloc(contracts.Energy.Address, &big.Int{}, contracts.Energy.RuntimeBytecodes()).
		Alloc(addr1, balance1, nil).
		Call(contracts.Energy.PackCharge(addr1, big.NewInt(1000000))).
		Build(state.NewCreator(kv))

	if err != nil {
		t.Fatal(err)
	}

	tx := new(tx.Builder).
		GasPrice(big.NewInt(1)).
		Gas(1000000).
		Clause(tx.NewClause(&addr2).WithValue(big.NewInt(10))).
		Build()

	sig, _ := crypto.Sign(tx.SigningHash().Bytes(), key)
	tx = tx.WithSignature(sig)

	state, _ := state.New(b0.Header().StateRoot(), kv)
	rt := runtime.New(state,
		thor.Address{}, 0, 0, 0, func(uint32) thor.Hash { return thor.Hash{} })
	receipt, _, err := rt.ExecuteTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	_ = receipt
	assert.Equal(t, state.GetBalance(addr1), new(big.Int).Sub(balance1, big.NewInt(10)))
}
