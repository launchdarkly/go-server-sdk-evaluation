//go:build launchdarkly_easyjson
// +build launchdarkly_easyjson

package ldmodel

import (
	"testing"

	"github.com/mailru/easyjson/jlexer"
	ej_jwriter "github.com/mailru/easyjson/jwriter"
)

func TestMarshalFlagEasyJSON(t *testing.T) {
	doMarshalFlagTest(t, func(flag FeatureFlag) ([]byte, error) {
		var writer ej_jwriter.Writer
		flag.MarshalEasyJSON(&writer)
		if writer.Error != nil {
			return nil, writer.Error
		}
		return writer.BuildBytes(nil)
	})
}

func TestMarshalSegmentEasyJSON(t *testing.T) {
	doMarshalSegmentTest(t, func(segment Segment) ([]byte, error) {
		var writer ej_jwriter.Writer
		segment.MarshalEasyJSON(&writer)
		if writer.Error != nil {
			return nil, writer.Error
		}
		return writer.BuildBytes(nil)
	})
}

func TestUnmarshalFlagEasyJSON(t *testing.T) {
	doUnmarshalFlagTest(t, func(data []byte) (FeatureFlag, error) {
		lexer := jlexer.Lexer{Data: data}
		var flag FeatureFlag
		flag.UnmarshalEasyJSON(&lexer)
		return flag, lexer.Error()
	})
}

func TestUnmarshalSegmentEasyJSON(t *testing.T) {
	doUnmarshalSegmentTest(t, func(data []byte) (Segment, error) {
		lexer := jlexer.Lexer{Data: data}
		var segment Segment
		segment.UnmarshalEasyJSON(&lexer)
		return segment, lexer.Error()
	})
}
