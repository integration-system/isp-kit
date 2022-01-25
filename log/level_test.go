package log_test

import (
	"testing"

	"github.com/integration-system/isp-kit/json"
	"github.com/integration-system/isp-kit/log"
	"github.com/stretchr/testify/require"
)

type someStruct struct {
	Level log.Level
}

func TestJson(t *testing.T) {
	require := require.New(t)

	s1 := someStruct{
		Level: log.FatalLevel,
	}
	data, err := json.Marshal(s1)
	require.NoError(err)

	s2 := someStruct{}
	err = json.Unmarshal(data, &s2)
	require.NoError(err)

	require.EqualValues(s1, s2)
}
