package block

import (
	"context"
	"errors"
	"fmt"
	"greet/consumer/internal/config"
	"greet/consumer/internal/svc"
	"greet/model/solmodel"
	"greet/pkg/constants"
	"net/http"
	"strings"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
	"github.com/panjf2000/ants/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

var ErrServiceStop = errors.New("service stop")

const SolChainIdInt = 100000

const ProgramStrPumpFun = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"

var ErrUnknownProgram = errors.New("unknown program")

type BlockService struct {
	Name string
	sc   *svc.ServiceContext
	c    *client.Client
	logx.Logger
	workerPool *ants.Pool
	Conn       *websocket.Conn
	ctx        context.Context
	cancel     func(err error)
	name       string
	slotChan   chan uint64
	solPrice   float64
}

func (s *BlockService) Stop() {
	s.cancel(ErrServiceStop)
	if s.Conn != nil {
		err := s.Conn.WriteMessage(websocket.TextMessage, []byte("{\"id\":1,\"jsonrpc\":\"2.0\",\"method\": \"blockUnsubscribe\", \"params\": [0]}\n"))
		if err != nil {
			s.Error("programUnsubscribe", err)
		}
		_ = s.Conn.Close()
	}
}

func (s *BlockService) Start() {
	s.GetBlockFromHttp()
}

func NewBlockService(sc *svc.ServiceContext, name string, slotChan chan uint64, index int) *BlockService {
	fmt.Println("name:", name)
	ctx, cancel := context.WithCancelCause(context.Background())
	pool, _ := ants.NewPool(5)
	solService := &BlockService{
		c: client.New(rpc.WithEndpoint(config.FindChainRpcByChainId(SolChainIdInt)), rpc.WithHTTPClient(&http.Client{
			Timeout: 5 * time.Second,
		})),
		sc:         sc,
		Logger:     logx.WithContext(context.Background()).WithFields(logx.Field("service", fmt.Sprintf("%s-%v", name, index))),
		workerPool: pool,
		ctx:        ctx,
		cancel:     cancel,
		name:       name,
		slotChan:   slotChan,
	}
	return solService
}

func (s *BlockService) GetBlockFromHttp() {
	ctx := s.ctx
	for {
		select {
		case <-s.ctx.Done():
			return
		case slot, ok := <-s.slotChan:
			if !ok {
				fmt.Println("current channel was closed")
				return
			}

			threading.RunSafe(func() {
				s.ProcessBlock(ctx, int64(slot))
			})

		}
	}
}

func (s *BlockService) ProcessBlock(ctx context.Context, slot int64) {
	if slot == 0 {
		return
	}

	block, err := s.sc.BlockModel.FindOneBySlot(ctx, slot)
	switch {
	case errors.Is(err, solmodel.ErrNotFound):
		block = &solmodel.Block{
			Slot: slot,
		}
	case err == nil:
		// Block found, check if already processed
		if block.Status == constants.BlockProcessed || block.Status == constants.BlockSkipped {
			s.Infof("processBlock:%v skip decode, block already processed: %#v", slot, block)
			return
		}
		// Continue processing the block
	default:
		s.Errorf("processBlock:%v findOneBySlot: %v, error: %v", slot, slot, err)
		return
	}

	blockInfo, err := GetSolBlockInfoDelay(s.sc.GetSolClient(), ctx, uint64(slot))
	if err != nil || blockInfo == nil {
		if err != nil && strings.Contains(err.Error(), "was skipped") {
			block.Status = constants.BlockSkipped
			// Set a default block time if not available
			s.Infof("processBlock:%v getSolBltants.BlockSkockInfo was skipped, err: %v", slot, err)
			_ = s.sc.BlockModel.Insert(ctx, block)
			return
		}
		// 异常区块记录，后续做兜底策略，把丢的区块补回来
		// Set a default block time if not available
		block.Status = constants.BlockFailed
		if block.BlockTime.IsZero() {
			block.BlockTime = time.Now()
		}
		err = s.sc.BlockModel.Insert(ctx, block)
		s.Errorf("processBlock:%v getSolBlockInfo error: %v", slot, err)
		return
	}

	// Set the block time from the retrieved block info
	if blockInfo.BlockTime != nil {
		block.BlockTime = *blockInfo.BlockTime
	} else {
		// Fallback to current time if blockInfo.BlockTime is nil
		block.BlockTime = time.Now()
	}

	if blockInfo.BlockHeight != nil {
		block.BlockHeight = *blockInfo.BlockHeight
	}
	block.Status = constants.BlockProcessed

	var tokenAccountMap = make(map[string]*TokenAccount)
	solPrice := s.GetBlockSolPrice(ctx, blockInfo, tokenAccountMap)
	if solPrice == 0 {
		s.Error("Sol Price not found")
		return
	}
	fmt.Println("sol price is:", solPrice)
	block.SolPrice = solPrice

	trades := make([]*TradeWithPair, 0, 1000)
	slice.ForEach[client.BlockTransaction](blockInfo.Transactions, func(index int, tx client.BlockTransaction) {

		decodeTx := &DecodedTx{
			BlockDb:         block,
			Tx:              &tx,
			TxIndex:         index,
			SolPrice:        solPrice,
			TokenAccountMap: tokenAccountMap,
		}

		trade, _ := DecodeTx(ctx, s.sc, decodeTx)

		trades = append(trades, trade...)
	})

	tradeMap := make(map[string][]*TradeWithPair)

	for _, trade := range trades {
		if len(trade.PairAddr) > 0 {
			tradeMap[trade.PairAddr] = append(tradeMap[trade.PairAddr], trade)
		}
	}

	//存储
	group := threading.NewRoutineGroup()

	group.RunSafe(func() {
		s.SaveTrades(ctx, constants.SolChainIdInt, tradeMap)
	})

	err = s.sc.BlockModel.Insert(ctx, block)
	if err != nil {
		fmt.Println("block insert err:", err)
		return
	}
	fmt.Println("Block insert successfully!, the corresponding slot is:", slot)
}

func DecodeTx(ctx context.Context, sc *svc.ServiceContext, dtx *DecodedTx) (trades []*TradeWithPair, err error) {
	if dtx.Tx == nil || dtx.BlockDb == nil {
		return
	}
	tx := dtx.Tx
	dtx.TxHash = base58.Encode(tx.Transaction.Signatures[0])

	if tx.Meta.Err != nil {
		return
	}

	dtx.InnerInstructionMap = GetInnerInstructionMap(tx)

	if len(tx.Meta.LogMessages) == 0 {
		return nil, fmt.Errorf("decode tx maybe vote tx, tx hash: %v", dtx.TxHash)
	}

	//遍历交易的所有指令
	for i := range tx.Transaction.Message.Instructions {
		instruction := &tx.Transaction.Message.Instructions[i]
		trade, err := DecodeInstruction(ctx, sc, dtx, instruction, i)
		if err == nil && trade != nil {
			trades = append(trades, trade)
			continue
		}
		if err != nil && !errors.Is(err, ErrTokenAmountIsZero) && !errors.Is(err, ErrNotSupportWarp) && !errors.Is(err, ErrNotSupportInstruction) {
			err = fmt.Errorf("decodeInstruction err:%v", err)
			logx.Error(err)
		}
	}
	return
}

func DecodeInstruction(ctx context.Context, sc *svc.ServiceContext, dtx *DecodedTx, instruction *types.CompiledInstruction, index int) (trade *TradeWithPair, err error) {
	tx := dtx.Tx
	program := tx.AccountKeys[instruction.ProgramIDIndex].String()

	if program == ProgramStrPumpFun {
		trade, err = DecodePumpInstruction(ctx, sc, dtx, instruction, index)
		return trade, err
	}
	return
}
