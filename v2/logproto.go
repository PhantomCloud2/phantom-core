package v2

import (
	"fmt"

	pb "internal-libcore/corerpc"
	"github.com/sagernet/sing/common/observable"
)

func NewObserver[T any](listenerBufferSize int) *observable.Observer[T] {
	return observable.NewObserver(&observable.Subscriber[T]{}, listenerBufferSize)
}

var logObserver = NewObserver[pb.LogMessage](10)

func Log(level pb.LogLevel, typ pb.LogType, message string) {
	if level != pb.LogLevel_DEBUG {
		fmt.Printf("%s %s %s\n", level, typ, message)
	}

	logObserver.Emit(pb.LogMessage{
		Level:   level,
		Type:    typ,
		Message: message,
	})
}
