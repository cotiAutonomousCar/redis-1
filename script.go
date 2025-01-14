package redis

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"
)

type Scripter interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *Cmd
	ScriptExists(ctx context.Context, hashes ...string) *BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *StringCmd
}

var (
	_ Scripter = (*Client)(nil)
	_ Scripter = (*Ring)(nil)
	_ Scripter = (*ClusterClient)(nil)
)

type Script struct {
	src, hash string
}

func NewScript(src string) *Script {
	h := sha1.New()
	_, _ = io.WriteString(h, src)
	return &Script{
		src:  src,
		hash: hex.EncodeToString(h.Sum(nil)),
	}
}

func (s *Script) Hash() string {
	return s.hash
}

func (s *Script) Load(ctx context.Context, c Scripter) *StringCmd {
	return c.ScriptLoad(ctx, s.src)
}

func (s *Script) Exists(ctx context.Context, c Scripter) *BoolSliceCmd {
	return c.ScriptExists(ctx, s.hash)
}

func (s *Script) Eval(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.Eval(ctx, s.src, keys, args...)
}

func (s *Script) EvalSha(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	return c.EvalSha(ctx, s.hash, keys, args...)
}

// Run optimistically uses EVALSHA to run the script. If script does not exist
// it is retried using EVAL.
func (s *Script) Run(ctx context.Context, c Scripter, keys []string, args ...interface{}) *Cmd {
	r := s.EvalSha(ctx, c, keys, args...)
	if err := r.Err(); err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		return s.Eval(ctx, c, keys, args...)
	}
	return r
}
