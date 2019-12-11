package blockchain

import (
	"fmt"
	"github.com/33cn/chain33/util"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func InitEnv() *BlockChain {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := queue.New("channel")
	q.SetConfig(cfg)
	chain := New(cfg)
	chain.client = q.Client()
	return chain
}

func TestGetStoreUpgradeMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	require.NotNil(t, blockStore)

	meta, err := blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "0.0.0")

	meta.Version = "1.0.0"
	err = blockStore.SetStoreUpgradeMeta(meta)
	require.NoError(t, err)
	meta, err = blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "1.0.0")
}

func TestSeqSaveAndGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = true
	blockStore.isParaChain = false

	newBatch := blockStore.NewBatch(true)
	seq, err := blockStore.saveBlockSequence(newBatch, []byte("s0"), 0, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	newBatch = blockStore.NewBatch(true)
	seq, err = blockStore.saveBlockSequence(newBatch, []byte("s1"), 1, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	s, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s)

	s2, err := blockStore.GetBlockSequence(s)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s1"), s2.Hash)

	s3, err := blockStore.GetSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s3)
}

func TestParaSeqSaveAndGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	bchain := InitEnv()
	blockStore := NewBlockStore(bchain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = true
	blockStore.isParaChain = true

	newBatch := blockStore.NewBatch(true)
	seq, err := blockStore.saveBlockSequence(newBatch, []byte("s0"), 0, 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	newBatch = blockStore.NewBatch(true)
	seq, err = blockStore.saveBlockSequence(newBatch, []byte("s1"), 1, 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	s, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s)

	s2, err := blockStore.GetBlockSequence(s)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s1"), s2.Hash)

	s3, err := blockStore.GetSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s3)

	s4, err := blockStore.GetMainSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s4)

	s5, err := blockStore.LoadBlockLastMainSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s5)

	s6, err := blockStore.GetBlockByMainSequence(1)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s0"), s6.Hash)

	chain := &BlockChain{
		blockStore: blockStore,
	}
	s7, err := chain.ProcGetMainSeqByHash([]byte("s0"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s7)

	_, err = chain.ProcGetMainSeqByHash([]byte("s0-not-exist"))
	assert.NotNil(t, err)
}

func TestSeqCreateAndDelete(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = false
	blockStore.isParaChain = true

	batch := blockStore.NewBatch(true)
	for i := 0; i <= 100; i++ {
		var header types.Header
		h0 := calcHeightToBlockHeaderKey(int64(i))
		header.Hash = []byte(fmt.Sprintf("%d", i))
		types.Encode(&header)
		batch.Set(h0, types.Encode(&header))
	}
	blockStore.height = 100
	batch.Write()

	blockStore.saveSequence = true
	blockStore.CreateSequences(10)
	seq, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(100), seq)

	seq, err = blockStore.GetSequenceByHash([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)

	seq, err = blockStore.GetSequenceByHash([]byte("0"))
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
}

func TestHasTx(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = false
	blockStore.isParaChain = false
	cfg.S("quickIndex", true)

	//txstring1 和txstring2的短hash是一样的，但是全hash是不一样的
	txstring1 := "0xaf095d11326ebb97d142fdb0e0138ef28524470c121b4811bdd05857b2d06764"
	txstring2 := "0xaf095d11326ebb97d142fdb0e0138ef28524470c121b4811bdd05857b2d06765"
	txstring3 := "0x8fac317e02ee25b1bbc5bd5a8570962b482928b014d14817b3c7a4e6aeddb3c6"
	txstring4 := "0x6522279c4fae53965e7bfbd35651dcd68813a50c65bf7af20b02c9bfe3d2ce8b"

	txhash1, err := common.FromHex(txstring1)
	assert.Nil(t, err)
	txhash2, err := common.FromHex(txstring2)
	assert.Nil(t, err)
	txhash3, err := common.FromHex(txstring3)
	assert.Nil(t, err)
	txhash4, err := common.FromHex(txstring4)
	assert.Nil(t, err)

	batch := blockStore.NewBatch(true)

	var txresult types.TxResult
	txresult.Height = 1
	txresult.Index = int32(1)
	batch.Set(cfg.CalcTxKey(txhash1), types.Encode(&txresult))
	batch.Set(types.CalcTxShortKey(txhash1), []byte("1"))

	txresult.Height = 3
	txresult.Index = int32(3)
	batch.Set(cfg.CalcTxKey(txhash3), types.Encode(&txresult))
	batch.Set(types.CalcTxShortKey(txhash3), []byte("1"))

	batch.Write()

	has, _ := blockStore.HasTx(txhash1)
	assert.Equal(t, has, true)

	has, _ = blockStore.HasTx(txhash2)
	assert.Equal(t, has, false)

	has, _ = blockStore.HasTx(txhash3)
	assert.Equal(t, has, true)

	has, _ = blockStore.HasTx(txhash4)
	assert.Equal(t, has, false)
}

func TestInitReduceLocaldb(t *testing.T) {

	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	//cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)

	// for test initReduceLocaldb
	flagHeight := int64(0)
	endHeight  := int64(80000)
	flag := int64(0)
	if flag == 0 {
		if endHeight > flagHeight {
			blockStore.reduceLocaldb(flagHeight, endHeight, false,
				func(batch dbm.Batch, height int64) {
				    batch.Set([]byte(fmt.Sprintf("key-%d", height)), []byte(fmt.Sprintf("value-%d", height)))
			    },
				func(batch dbm.Batch, height int64) {
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data:height}))
				})
			// CompactRange执行将会阻塞仅仅做一次压缩
			chainlog.Info("reduceLocaldb start compact db")
			blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db")
		}
		blockStore.saveReduceLocaldbFlag()
	}

	flag, err = blockStore.loadFlag(types.FlagReduceLocaldb)
	assert.NoError(t, err)
	assert.Equal(t, flag, int64(1))

	flagHeight, err = blockStore.loadFlag(types.ReduceLocaldbHeight)
	assert.NoError(t, err)
	assert.Equal(t, flagHeight, endHeight)

}

func TestInitReduceLocaldb1(t *testing.T) {

	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	//cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)

	// for test initReduceLocaldb
	flagHeight := int64(0)
	endHeight  := int64(80000)
	flag := int64(0)
	if flag == 0 {
		defer func() {
			if r := recover(); r != nil {
				flag, err = blockStore.loadFlag(types.FlagReduceLocaldb)
				assert.NoError(t, err)
				assert.Equal(t, flag, int64(0))

				flagHeight, err = blockStore.loadFlag(types.ReduceLocaldbHeight)
				assert.NoError(t, err)
				assert.NotEqual(t, flagHeight, int64(endHeight))
				return
			}
		}()

		if endHeight > flagHeight {
			blockStore.reduceLocaldb(flagHeight, endHeight, false,
				func(batch dbm.Batch, height int64) {
					batch.Set([]byte(fmt.Sprintf("key-%d", height)), []byte(fmt.Sprintf("value-%d", height)))
				},
				func(batch dbm.Batch, height int64) {
					if height == endHeight {
						panic("for test")
					}
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data:height}))
				})
			// CompactRange执行将会阻塞仅仅做一次压缩
			chainlog.Info("reduceLocaldb start compact db")
			blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db")
		}
		blockStore.saveReduceLocaldbFlag()
	}
}

func TestReduceBody(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)

	// generate blockdetail
	txs := util.GenCoinsTxs(cfg, util.HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01"), 10)
	block := &types.Block{Txs:txs}
	block.MainHash = block.Hash(cfg)
	block.Height = 0
	blockdetail := &types.BlockDetail{
		Block: block,
		Receipts: []*types.ReceiptData{
			{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
			{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
			{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
			{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
		},
		KV: []*types.KeyValue{{Key: []byte("000"), Value: []byte("000")}, {Key: []byte("111"), Value: []byte("111")}},
	}

	// save blockdetail
	newbatch := blockStore.NewBatch(true)
	_, err = blockStore.SaveBlock(newbatch, blockdetail, 0)
	assert.NoError(t, err)
	newbatch.Write()

	// reduceBody
	newbatch = blockStore.NewBatch(true)
	blockStore.reduceBody(newbatch, 0)
	newbatch.Write()

	// check
	body, err := blockStore.loadBlockBody(0)
	assert.NoError(t, err)
	for _, recep := range body.Receipts {
		assert.Nil(t, recep.Logs)
	}
}

func TestReduceBodyInit(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)


	// generate blockdetail
	txs := util.GenCoinsTxs(cfg, util.HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01"), 10)
	block := &types.Block{Txs:txs}
	block.MainHash = block.Hash(cfg)
	block.Height = 0
	blockdetail := &types.BlockDetail{
		Block: block,
		Receipts: []*types.ReceiptData{
			{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
			{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
			{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
			{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
		},
		KV: []*types.KeyValue{{Key: []byte("000"), Value: []byte("000")}, {Key: []byte("111"), Value: []byte("111")}},
	}

	// save blockdetail
	newbatch := blockStore.NewBatch(true)
	_, err = blockStore.SaveBlock(newbatch, blockdetail, 0)
	assert.NoError(t, err)
	newbatch.Write()

	// save tx TxResult
	newbatch = blockStore.NewBatch(true)
	for index, tx := range txs  {
		var txresult types.TxResult
		txresult.Height = block.Height
		txresult.Index = int32(index)
		txresult.Tx = tx
		txresult.Receiptdate = &types.ReceiptData{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}}
		txresult.Blocktime = 3123131231
		txresult.ActionName = tx.ActionName()
		newbatch.Set(cfg.CalcTxKey(tx.Hash()), cfg.CalcTxKeyValue(&txresult))
	}
	newbatch.Write()

	// reduceBodyInit
	cfg.S("reduceLocaldb", true)
	newbatch = blockStore.NewBatch(true)
	blockStore.reduceBodyInit(newbatch, 0)
	newbatch.Write()

	// check
	// 1 body
	body, err := blockStore.loadBlockBody(0)
	assert.NoError(t, err)
	for _, recep := range body.Receipts {
		assert.Nil(t, recep.Logs)
	}
	// 2 tx
	for _, tx := range txs  {
		hash := tx.Hash()
		_, err := blockStore.db.Get(hash)
		assert.Error(t, err, types.ErrNotFound)
		v, err := blockStore.db.Get(cfg.CalcTxKey(hash))
		assert.NoError(t, err)
		var txresult types.TxResult
		err = types.Decode(v, &txresult)
		assert.NoError(t, err)
		assert.Nil(t, txresult.Receiptdate)
	}
}

func TestGetRealTxResult(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)

	// generate blockdetail
	txs := util.GenCoinsTxs(cfg, util.HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01"), 10)
	block := &types.Block{Txs:txs}
	block.MainHash = block.Hash(cfg)
	block.Height = 0
	blockdetail := &types.BlockDetail{
		Block: block,
		Receipts: []*types.ReceiptData{
			{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
			{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
			{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
			{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
		},
		KV: []*types.KeyValue{{Key: []byte("000"), Value: []byte("000")}, {Key: []byte("111"), Value: []byte("111")}},
	}

	// save blockdetail
	newbatch := blockStore.NewBatch(true)
	_, err = blockStore.SaveBlock(newbatch, blockdetail, 0)
	assert.NoError(t, err)
	newbatch.Write()

	// check
	cfg.S("reduceLocaldb", true)
	txr := &types.TxResult{
		Height: 0,
		Index: 0,
	}
	blockStore.getRealTxResult(txr)
	assert.Equal(t, txr.Tx.Nonce, txs[0].Nonce)
	assert.Equal(t, txr.Receiptdate.Ty, blockdetail.Receipts[0].Ty)
}