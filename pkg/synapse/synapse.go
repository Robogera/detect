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
	Subject   string   `json:"subject"`
	Message   *Message `json:"message"`
}

type Message struct {
	People []*person.ExportedPerson `json:"people"`
}

func (c *Event) ToPayload() ([]byte, error) {
	return json.Marshal(c)
}
