package notifier

import (
	"github.com/groob/plist"
)

type MDMCommandPayload struct {
	RequestType string
	Data        []byte `plist:",omitempty"`
}

type MDMCommand struct {
	Command     MDMCommandPayload
	CommandUUID string
}

func NewMDMCommand(uuid string, data []byte) ([]byte, error) {
	c := &MDMCommand{
		Command: MDMCommandPayload{
			RequestType: "DeclarativeManagement",
			Data:        data,
		},
		CommandUUID: uuid,
	}
	return plist.Marshal(c)
}
