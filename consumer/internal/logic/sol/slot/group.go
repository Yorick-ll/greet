package slot

import (
	"greet/consumer/internal/svc"
)

type SlotServiceGroup struct {
	*SlotService
	Ws           *SlotWsService
	NotCompleted *SloteNotCompleteService
}

func NewSlotServiceGroup(sc *svc.ServiceContext, slotChan, errChan chan uint64) *SlotServiceGroup {
	slotService := NewSlotService(sc, slotChan, errChan)
	return &SlotServiceGroup{
		SlotService:  slotService,
		Ws:           NewSlotWsService(slotService),
		NotCompleted: NewSlotNotCompleteService(slotService),
	}
}

func (s *SlotServiceGroup) Start() {
	go s.NotCompleted.Start()
	s.Ws.Start()
}
