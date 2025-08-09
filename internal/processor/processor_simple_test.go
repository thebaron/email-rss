package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProcessor(t *testing.T) {
	proc := New(nil, nil, nil)

	assert.NotNil(t, proc)
	assert.Nil(t, proc.imapClient)
	assert.Nil(t, proc.database)
	assert.Nil(t, proc.rssGenerator)
	assert.Nil(t, proc.aiHooks)
}

func TestSetAIHooks(t *testing.T) {
	proc := New(nil, nil, nil)

	hooks := &stubAIHooks{}
	proc.SetAIHooks(hooks)

	assert.Equal(t, hooks, proc.aiHooks)
}

type stubAIHooks struct{}

func (s *stubAIHooks) SummarizeMessage(subject, body string) (string, error) {
	return "Summary: " + subject, nil
}
