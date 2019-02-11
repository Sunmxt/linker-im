package gate

import (
	"time"
)

func (g *Gate) Keepalive() {
	for {
		g.LB.Keepalive()
		time.Sleep(time.Second)
	}
}
