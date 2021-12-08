package rmq

type Observer interface {
	ClientReady()
	ClientError(err error)
	ConsumerError(consumer Consumer, err error)
	ShutdownStarted()
	ShutdownDone()
}

type NoopObserver struct {
}

func (n NoopObserver) ClientReady() {

}

func (n NoopObserver) ClientError(err error) {

}

func (n NoopObserver) ConsumerError(consumer Consumer, err error) {

}

func (n NoopObserver) ShutdownStarted() {
}

func (n NoopObserver) ShutdownDone() {
}
