package signal

import (
	"os"
	"os/signal"
	"syscall"
)

func Watch() chan struct{} {
	hup := make(chan struct{}, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		<-sigs
		hup <- struct{}{}
	}()

	return hup
}