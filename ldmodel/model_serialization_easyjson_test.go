// +build launchdarkly_easyjson

package ldmodel

import (
	"testing"

	"github.com/mailru/easyjson/jlexer"
	ej_jwriter "github.com/mailru/easyjson/jwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalFlagEasyJSON(t *testing.T) {
	var writer ej_jwriter.Writer
	flagWithAllProperties.MarshalEasyJSON(&writer)
	require.NoError(t, writer.Error)
	bytes, err := writer.BuildBytes(nil)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, flagWithAllPropertiesJSON, json)
}

func TestMarshalSegmentEasyJSON(t *testing.T) {
	var writer ej_jwriter.Writer
	segmentWithAllProperties.MarshalEasyJSON(&writer)
	require.NoError(t, writer.Error)
	bytes, err := writer.BuildBytes(nil)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, segmentWithAllPropertiesJSON, json)
}

func TestUnmarshalFlagEasyJSON(t *testing.T) {
	bytes := toJSON(flagWithAllPropertiesJSON)
	lexer := jlexer.Lexer{Data: bytes}
	var flag FeatureFlag
	flag.UnmarshalEasyJSON(&lexer)
	require.NoError(t, lexer.Error())
	assert.Equal(t, flagWithAllProperties, flag)
}

func TestUnmarshalSegmentEasyJSON(t *testing.T) {
	bytes := toJSON(segmentWithAllPropertiesJSON)
	lexer := jlexer.Lexer{Data: bytes}
	var segment Segment
	segment.UnmarshalEasyJSON(&lexer)
	require.NoError(t, lexer.Error())
	assert.Equal(t, segmentWithAllProperties, segment)
}
