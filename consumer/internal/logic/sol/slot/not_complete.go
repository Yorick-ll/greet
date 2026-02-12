package slot

import (
	"errors"
	"fmt"
	"greet/model/solmodel"
	"time"
)

type SlotNotCompleteService struct {
	*SlotService
}

func NewSlotNotCompleteService(SlotService *SlotService) *SlotNotCompleteService {
	return &SlotNotCompleteService{
		SlotService: SlotService,
	}
}

func (s *SlotNotCompleteService) Start() {
	s.SlotNotCompleted()
}

func (s *SlotService) SlotNotCompleted() {
	fmt.Println("fail block processiong !!!")

	slot := s.sc.Config.Sol.StartBlock

	// 如果slot为0，则从数据库中获取第一个失败的区块
	if slot == 0 {
		block, err := s.sc.BlockModel.FindFirstFailedBlock(s.ctx)

		if err != nil {
			slot = 0
			return
		} else {
			slot = uint64(block.Slot)
		}
	}

	fmt.Println("The first failed block is:", slot)
	// 每5秒检查一次
	var checkTicker = time.NewTicker(time.Millisecond * 5000)
	// 每一秒发送一次
	var sendTicker = time.NewTicker(time.Millisecond * 1000)
	// 关闭定时器
	defer checkTicker.Stop()
	defer sendTicker.Stop()

	for {
		select {
		// 上下文关闭
		case <-s.ctx.Done():
			return
		case <-checkTicker.C:
			slots, err := s.sc.BlockModel.FindProcessingSlots(s.ctx, int64(slot-100), 50)

			switch {
			case errors.Is(err, solmodel.ErrNotFound) || len(slots) == 0:
				return
			case err == nil:
			default:
				s.Error("FindProcessingSlot error: %v", err)
			}

			fmt.Println("The number of processing slots is:", len(slots))

			for _, slot := range slots {
				select {
				case <-s.ctx.Done():
					return
				case <-sendTicker.C:
					s.errorCh <- uint64(slot.Slot)
				}

			}
		}

	}

}
