package rmq

import (
	"github.com/streadway/amqp"
)

type Declarations struct {
	Exchanges []Exchange
	Queues    []Queue
	Bindings  []Binding
}

type Queue struct {
	Name string
	Args amqp.Table
}

type Exchange struct {
	Name string
	Type string
	Args amqp.Table
}

type Binding struct {
	ExchangeName string
	QueueName    string
	RoutingKey   string
	Args         amqp.Table
}
