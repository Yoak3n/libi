package dispatch

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
