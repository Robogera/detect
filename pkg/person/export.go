package person

type ExportedPerson struct {
	Id string `json:"id"`
	X  uint   `json:"x"`
	Y  uint   `json:"y"`
}

func (p *Person) Export() *ExportedPerson {
	return &ExportedPerson{
		Id: p.Id(),
		X:  uint(p.State().X),
		Y:  uint(p.State().Y),
	}
}
