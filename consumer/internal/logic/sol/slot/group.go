package slot

import (
	"greet/consumer/internal/svc"
)

type SlotServiceGroup struct {
	*SlotService
	Ws *SlotWsService
}

func NewSlotServiceGroup(sc *svc.ServiceContext, realChan chan uint64) *SlotServiceGroup {
	slotService := NewSlotService(sc, realChan)
	return &SlotServiceGroup{
		SlotService: slotService,
		Ws:          NewSlotWsService(slotService),
	}
}

func (s *SlotServiceGroup) Start() {
	s.Ws.Start()
}
