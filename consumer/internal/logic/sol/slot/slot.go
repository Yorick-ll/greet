package slot

import (
	"context"
	"errors"

	"greet/consumer/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var ErrServiceStop = errors.New("service stop")

type SlotService struct {
	Conn *websocket.Conn
	sc   *svc.ServiceContext
	logx.Logger
	ctx        context.Context
	cancel     func(err error)
	maxSlot    uint64
	realtimeCh chan uint64
	errorCh    chan uint64
}

func NewSlotService(sc *svc.ServiceContext, slotChan, errChan chan uint64) *SlotService {
	ctx, cancel := context.WithCancelCause(context.Background())
	return &SlotService{
		Logger:     logx.WithContext(context.Background()).WithFields(logx.Field("service", "slot")),
		sc:         sc,
		ctx:        ctx,
		cancel:     cancel,
		realtimeCh: slotChan,
		errorCh:    errChan,
	}
}

func (s *SlotService) Start() {
}

func (s *SlotService) Stop() {
	s.Info("stop slot")
	s.cancel(ErrServiceStop)
	if s.Conn != nil {
		err := s.Conn.WriteMessage(websocket.TextMessage, []byte("{\"id\":1,\"jsonrpc\":\"2.0\",\"method\": \"slotUnsubscribe\", \"params\": [0]}\n"))
		if err != nil {
			s.Error("programUnsubscribe", err)
		}
		_ = s.Conn.Close()
	}
}
