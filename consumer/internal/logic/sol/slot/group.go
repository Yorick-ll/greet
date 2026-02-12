package slot

import (
	"greet/consumer/internal/svc"

	"github.com/zeromicro/go-zero/core/threading"
)

type SlotServiceGroup struct {
	*SlotService
	Ws          *SlotWsService
	NotComplete *SlotNotCompleteService
}

func NewSlotServiceGroup(sc *svc.ServiceContext, realChan chan uint64) *SlotServiceGroup {
	slotService := NewSlotService(sc, realChan)
	return &SlotServiceGroup{
		SlotService: slotService,
		Ws:          NewSlotWsService(slotService),
		NotComplete: NewSlotNotCompleteService(slotService),
	}
}

func (s *SlotServiceGroup) Start() {
	threading.GoSafe(func() {
		s.NotComplete.Start()
	})
	s.Ws.Start()
}
