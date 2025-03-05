package person

type Status string

const (
	STATUS_EXPIRED    Status = "expired"
	STATUS_OOB        Status = "out of bounds"
	STATUS_LOST       Status = "lost"
	STATUS_ASSOCIATED Status = "associated"
	STATUS_NEW        Status = "new"
	STATUS_VALIDATED  Status = "validated"
)
