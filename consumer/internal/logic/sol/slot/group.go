package slot

import (
	"greet/consumer/internal/svc"
)

type SlotServiceGroup struct {
	*SlotService
	Ws *SlotWsService
}

func NewSlotServiceGroup(sc *svc.ServiceContext) *SlotServiceGroup {
	slotService := NewSlotService(sc)
	return &SlotServiceGroup{
		SlotService: slotService,
		Ws:          NewSlotWsService(slotService),
	}
}

func (s *SlotServiceGroup) Start() {
	s.Ws.Start()
}
