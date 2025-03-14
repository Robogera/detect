package synapse

import (
	"encoding/json"

	"github.com/Robogera/detect/pkg/person"
)

type Event struct {
	Id        uint     `json:"id"`
	Sender    string   `json:"sender"`
	Type      string   `json:"type"`
	Initiator string   `json:"initiator"`
	Receiver  string   `json:"receiver"`
	Message   *Message `json:"message"`
}

type Message struct {
	Subject    string      `json:"subject"`
	Parameters *Parameters `json:"parameters"`
}

type Parameters struct {
	Detections []*person.ExportedPerson `json:"detections"`
}

func (c *Event) ToPayload() ([]byte, error) {
	return json.Marshal(c)
}
