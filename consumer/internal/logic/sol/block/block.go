package block

import (
	"context"
	"errors"
	"fmt"
	"greet/consumer/internal/svc"
	"greet/model/solmodel"
	"greet/pkg/constants"
	"net/http"
	"strings"
	"time"

	"greet/consumer/internal/config"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gorilla/websocket"
	"github.com/panjf2000/ants/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

var ErrServiceStop = errors.New("service stop")

const ProgramStrPumpFun = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"

var ErrUnknownProgram = errors.New("unknown program")

type BlockService struct {
	Name string
	sc   *svc.ServiceContext
	c    *client.Client
	logx.Logger
	workerPool *ants.Pool
	slotChan   chan uint64
	Conn       *websocket.Conn
	ctx        context.Context
	cancel     func(err error)
	name       string
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
	// 获取区块
	s.GetBlockFromHttp()
}

func NewBlockService(sc *svc.ServiceContext, name string, slotChan chan uint64, index int) *BlockService {
	fmt.Println("name:", name)
	ctx, cancel := context.WithCancelCause(context.Background())
	pool, _ := ants.NewPool(5)
	solService := &BlockService{
		c: client.New(rpc.WithEndpoint(config.FindChainRpcByChainId(constants.SolChainIdInt)), rpc.WithHTTPClient(&http.Client{
			Timeout: 5 * time.Second,
		})),
		sc:         sc,
		Logger:     logx.WithContext(context.Background()).WithFields(logx.Field("service", fmt.Sprintf("%s-%v", name, index))),
		slotChan:   slotChan,
		workerPool: pool,
		ctx:        ctx,
		cancel:     cancel,
		name:       name,
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
				fmt.Print("***********")
				return
			}
			//打印当前最新slot
			// fmt.Println("current slot is:", slot)

			threading.GoSafe(func() {
				s.ProcessBlock(ctx, int64(slot))
			})
		}
	}
}

func (s *BlockService) ProcessBlock(ctx context.Context, slot int64) {
	if slot == 0 {
		return
	}
	// 查询当前slot是否被处理过
	block, err := s.sc.BlockModel.FindOneBySlot(ctx, slot)
	switch {
	case err != nil && strings.Contains(err.Error(), "record not found"):
		block = &solmodel.Block{
			Slot: slot,
		}
	default:
		s.Errorf("processBlock:%v findOneBySlot: %v, error: %v", slot, slot, err)
		return
	}

	blockInfo, err := GetSolBlockInfoDelay(s.sc.GetSolClient(), ctx, uint64(slot))
	if err == nil || blockInfo == nil {
		if err != nil && strings.Contains(err.Error(), "was skipped") {
			block.Status = constants.BlockSkipped

			s.Infof("processBlock:%v getSolBltants.BlockSkockInfo was skipped, err: %v", slot, err)
			_ = s.sc.BlockModel.Insert(ctx, block)
			return
		}
		// 异常区块记录，后续做兜底策略，把丢的区块补回来
		block.Status = constants.BlockFailed
		if block.BlockTime.IsZero() {
			block.BlockTime = time.Now()
		}
		err = s.sc.BlockModel.Insert(ctx, block)
		s.Errorf("processBlock:%v getSolBlockInfo error: %v", slot, err)
		return
	}
	// 设置区块时间
	if blockInfo.BlockTime != nil {
		block.BlockTime = *blockInfo.BlockTime
	} else {
		block.BlockTime = time.Now()
	}
	// 设置区块高度
	if blockInfo.BlockHeight != nil {
		block.BlockHeight = *blockInfo.BlockHeight
	}
	block.Status = constants.BlockProcessed

	slice.ForEach[client.BlockTransaction](blockInfo.Transactions, func(index int, tx client.BlockTransaction) {
		DeCodeTX(&tx)
	})

	err = s.sc.BlockModel.Insert(ctx, block)
	if err != nil {
		fmt.Println("block insert err:", err)
		return
	}
	fmt.Println("Block insert successfully!, the corresponding slot is:", slot)
}

func DeCodeTX(tx *client.BlockTransaction) {
	if tx == nil {
		return
	}
	// 遍历交易的所有指令
	for i := range tx.Transaction.Message.Instructions {
		instruction := &tx.Transaction.Message.Instructions[i]
		DeCodeInstruction(instruction, tx)
	}
}

func DeCodeInstruction(instruction *types.CompiledInstruction, tx *client.BlockTransaction) (err error) {

	if instruction == nil {
		return errors.New("instruction is nil")
	}

	if len(tx.AccountKeys) == 0 {
		return errors.New("account keys is empty")
	}

	program := tx.AccountKeys[instruction.ProgramIDIndex].String()

	if program == ProgramStrPumpFun {
		DecodePumpInstruction(instruction, tx)
	}

	return ErrUnknownProgram
}
