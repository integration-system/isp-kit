package http_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	isphttp "github.com/integration-system/isp-kit/http"
	"github.com/integration-system/isp-kit/http/endpoint"
	"github.com/integration-system/isp-kit/http/httperrors"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type Request struct {
	Id string `valid:"required"`
}

type Response struct {
	Result string
}

func TestService(t *testing.T) {
	url := prepareServer(t)
	response := Response{}

	client := resty.New().SetBaseURL("http://" + url)
	resp, err := client.R().
		SetBody(Request{Id: "man"}).
		SetResult(&response).
		Post("/getId")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	expected := Response{Result: "Hello_man"}
	require.Equal(t, expected, response)

	resp, err = client.R().
		SetBody(Request{Id: ""}).
		Post("/getId")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode())

	resp, err = client.R().
		SetBody(Request{Id: "smth"}).
		Post("/badGetId")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode())

	response = Response{}
	resp, err = client.R().
		SetResult(&response).
		Get("/noBody")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	expected = Response{Result: "Test"}
	require.Equal(t, expected, response)
}

type endpointDescriptor struct {
	Path    string
	Handler interface{}
}

func prepareServer(t *testing.T) string {
	logger, err := log.New(log.WithLevel(log.DebugLevel))
	require.NoError(t, err)

	endpoints := []endpointDescriptor{{
		Path: "/getId",
		Handler: func(req Request) (*Response, error) {
			return &Response{Result: "Hello_" + req.Id}, nil
		},
	}, {
		Path: "/badGetId",
		Handler: func(req Request) (*Response, error) {
			return &Response{}, httperrors.New(404, errors.New("Not Found"))
		},
	}, {
		Path: "/noBody",
		Handler: func(ctx context.Context) (*Response, error) {
			return &Response{Result: "Test"}, nil
		},
	}}

	mapper := endpoint.DefaultWrapper(logger)
	muxer := http.NewServeMux()
	for _, descriptor := range endpoints {
		muxer.Handle(descriptor.Path, mapper.Endpoint(descriptor.Handler))
	}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	srv := isphttp.NewServer()
	srv.Upgrade(muxer)
	go func() {
		err := srv.Serve(listener)
		require.NoError(t, err)
	}()

	return listener.Addr().String()
}