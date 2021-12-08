package endpoint

import (
	"context"
	"reflect"

	"github.com/integration-system/isp-kit/grpc"
	"github.com/integration-system/isp-kit/grpc/isp"
)

type Middleware func(next grpc.HandlerFunc) grpc.HandlerFunc

type RequestBodyExtractor interface {
	Extract(ctx context.Context, message *isp.Message, reqBodyType reflect.Type) (reflect.Value, error)
}

type ResponseBodyMapper interface {
	Map(result interface{}) (*isp.Message, error)
}

type ParamBuilder func(ctx context.Context, message *isp.Message) (interface{}, error)

type ParamMapper struct {
	Type    string
	Builder ParamBuilder
}

type Mapper struct {
	paramMappers  map[string]ParamMapper
	bodyExtractor RequestBodyExtractor
	bodyMapper    ResponseBodyMapper
	middlewares   []Middleware
}

func NewMapper(
	paramMappers []ParamMapper,
	bodyExtractor RequestBodyExtractor,
	bodyMapper ResponseBodyMapper,
) Mapper {
	mappers := make(map[string]ParamMapper)
	for _, mapper := range paramMappers {
		mappers[mapper.Type] = mapper
	}
	return Mapper{
		paramMappers:  mappers,
		bodyExtractor: bodyExtractor,
		bodyMapper:    bodyMapper,
	}
}

func (m Mapper) Endpoint(f interface{}) grpc.HandlerFunc {
	caller, err := NewCaller(f, m.bodyExtractor, m.bodyMapper, m.paramMappers)
	if err != nil {
		panic(err)
		return nil
	}

	handler := caller.Handle
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}
	return handler
}

func (m Mapper) WithMiddlewares(middlewares ...Middleware) Mapper {
	return Mapper{
		paramMappers:  m.paramMappers,
		bodyExtractor: m.bodyExtractor,
		bodyMapper:    m.bodyMapper,
		middlewares:   append(m.middlewares, middlewares...),
	}

}
