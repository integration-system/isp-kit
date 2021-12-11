package shutdown

import (
	"os"
	"os/signal"
)

func On(do func()) chan os.Signal {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		do()
	}()
	return ch
}
