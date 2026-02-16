package slot

import (
	"errors"
	"fmt"
	"greet/model/solmodel"
	"time"
)

type SloteNotCompleteService struct {
	*SlotService
}

func NewSlotNotCompleteService(SlotService *SlotService) *SloteNotCompleteService {
	return &SloteNotCompleteService{
		SlotService: SlotService,
	}
}

func (s *SloteNotCompleteService) Start() {
	s.SlotNotCompleted()
}

func (s *SlotService) SlotNotCompleted() {
	fmt.Println("failed block processing")
	slot := s.sc.Config.Sol.StartBlock

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

	var checkTicker = time.NewTicker(time.Millisecond * 5000)
	var sendTicker = time.NewTicker(time.Millisecond * 1000)
	defer checkTicker.Stop()
	defer sendTicker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-checkTicker.C:
			slots, err := s.sc.BlockModel.FindProcessingSlots(s.ctx, int64(slot-100), 50)
			switch {
			case errors.Is(err, solmodel.ErrNotFound) || len(slots) == 0:
				return
			case err == nil:
			default:
				s.Error("FindProcessingSlot err:", err)
			}
			fmt.Println("The length of failed blocks is:", len(slots))

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
