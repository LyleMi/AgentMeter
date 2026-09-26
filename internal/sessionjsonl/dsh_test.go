package sessionjsonl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const dshSession = `{"type":"session","version":4,"id":"dsh-example","createdAt":1780000000000,"cwd":"/work/project","isSeeded":false,"delegationDepth":0}
{"type":"step/start","seq":0,"time":1780000001000,"data":{"turn":1,"step":1}}
{"type":"request/header","seq":1,"time":1780000001000,"data":{"header":{"config":{"provider":"deepseek","model":"deepseek-flash"}}}}
{"type":"assistant/chunk","seq":2,"time":1780000001500,"data":{"chunk":{"type":"usage","usage":{"inputTokens":100,"cacheReadTokens":200,"outputTokens":30}}}}
{"type":"assistant/message","seq":3,"time":1780000002000,"data":{"message":{"role":"assistant","content":[{"type":"text","text":"Done"}],"source":{"kind":"model","provider":"deepseek","model":"deepseek-flash"}},"usage":{"inputTokens":100,"cacheReadTokens":200,"cacheWriteTokens":10,"outputTokens":30,"reasoningTokens":20}}}
{"type":"tool/call","seq":4,"time":1780000002000,"data":{"callId":"call-1","name":"bash","arguments":"{\"command\":\"pwd\"}"}}
{"type":"tool/result","seq":5,"time":1780000002500,"data":{"message":{"role":"tool","toolCallId":"call-1","source":{"kind":"tool","callId":"call-1"},"content":[{"type":"text","text":"/work/project"}]}}}
`

func TestDSHPlainAndCompressed(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "session.v4.jsonl")
		data := []byte(dshSession)
		if compressed {
			path += ".zstd"
			enc, err := zstd.NewWriter(nil)
			if err != nil {
				t.Fatal(err)
			}
			data = enc.EncodeAll(data, nil)
			enc.Close()
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		parsed, hash, err := ParseFileWithHash(path, 1, 2)
		if err != nil {
			t.Fatal(err)
		}
		wantHash, err := HashFile(path)
		if err != nil || hash != wantHash {
			t.Fatalf("physical hash mismatch: %s %s %v", hash, wantHash, err)
		}
		again, err := ParseFile(path, 1, 2)
		if err != nil || again.Usage != parsed.Usage {
			t.Fatalf("plain read differs: %+v %v", again.Usage, err)
		}
		if parsed.Session.SessionKey != "dsh-example" || parsed.Session.ProjectPath != "/work/project" || parsed.Session.ModelProvider != "deepseek" || parsed.Session.ParseStatus != "ok" {
			t.Fatalf("session: %+v, warnings: %v", parsed.Session, parsed.Warnings)
		}
		if parsed.Usage.InputTokens != 310 || parsed.Usage.CachedInputTokens != 200 || parsed.Usage.TotalTokens != 340 || parsed.Usage.OutputTokens != 30 || len(parsed.ModelCall) != 1 {
			t.Fatalf("usage: %+v calls: %+v", parsed.Usage, parsed.ModelCall)
		}
		if parsed.ModelCall[0].DurationMS != 1000 || len(parsed.ToolCall) != 1 || parsed.ToolCall[0].DurationMS != 500 || parsed.ToolCall[0].ToolName != "bash" {
			t.Fatalf("calls: %+v %+v", parsed.ModelCall, parsed.ToolCall)
		}
	}
}

func TestDSHInheritedUsageAndUnknownModel(t *testing.T) {
	input := strings.Replace(dshSession, `"isSeeded":false`, `"isSeeded":true`, 1) + `{"type":"session/end-seed","seq":6,"time":1780000003000,"data":{"inherited":true}}
{"type":"step/start","seq":7,"time":1780000003000,"data":{}}
{"type":"assistant/message","seq":8,"time":1780000004000,"data":{"message":{"source":{"model":"custom-model","provider":"custom"}},"usage":{"inputTokens":20,"outputTokens":5},"interrupted":true}}
`
	parsed, err := parseFromReader("session.v4.jsonl", strings.NewReader(input), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.ModelCall) != 1 || len(parsed.ToolCall) != 0 || parsed.Usage.InputTokens != 20 || parsed.ModelCall[0].Model != "custom-model" || parsed.ModelCall[0].Status != "interrupted" {
		t.Fatalf("inherited usage counted: %+v", parsed)
	}
	for _, bad := range []string{strings.Replace(dshSession, `"version":4`, `"version":5`, 1), strings.Replace(dshSession, `"isSeeded":false`, `"isSeeded":true`, 1)} {
		if _, err := parseFromReader("session.jsonl", strings.NewReader(bad), 1, 2); err == nil {
			t.Fatal("unsupported/incomplete log should fail")
		}
	}
	unknown := `{"type":"session","version":0,"id":"unknown","createdAt":1000}
{"type":"assistant/message","seq":0,"time":2000,"data":{"usage":{"inputTokens":10,"outputTokens":2}}}
`
	parsed, err = parseFromReader("session.jsonl", strings.NewReader(unknown), 1, 2)
	if err != nil || parsed.ModelCall[0].Model != "unknown" {
		t.Fatalf("missing model must not invent a priced model: %+v %v", parsed, err)
	}
}

func TestDSHLegacySeedAndAttemptUsage(t *testing.T) {
	input := `{"type":"session","version":0,"id":"child","createdAt":10000,"seedLength":1,"parentSession":"parent"}
{"type":"assistant/message","seq":0,"time":1000,"data":{"usage":{"inputTokens":999,"outputTokens":99}}}
{"type":"step/start","seq":1,"time":11000,"data":{}}
{"type":"request/header","seq":2,"time":11000,"data":{"header":{"config":{"provider":"deepseek","model":"deepseek-flash"}}}}
{"type":"assistant/attempt","seq":3,"time":12000,"data":{"stream":[{"type":"chunk","time":11500,"chunk":{"type":"usage","usage":{"inputTokens":10,"outputTokens":1}}},{"type":"chunk","time":12000,"chunk":{"type":"usage","usage":{"inputTokens":10,"outputTokens":2}}}]}}
{"type":"tool/call","seq":4,"time":12000,"data":{"callId":"c","name":"bash","arguments":"{}"}}
{"type":"tool/result","seq":5,"time":13000,"data":{"message":{"role":"user","source":{"kind":"tool","callId":"c"},"content":[{"type":"tool-result","toolCallId":"c","isError":true,"content":[{"type":"text","text":"failed"}]}]}}}
{"type":"tool/result","seq":6,"time":14000,"surfaceOp":{"op":"replace","start":5,"end":5},"data":{"callId":"c","content":[]}}
`
	parsed, err := parseFromReader("session.jsonl", strings.NewReader(input), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.InputTokens != 10 || parsed.Usage.OutputTokens != 2 || len(parsed.ModelCall) != 1 || parsed.ModelCall[0].Status != "interrupted" {
		t.Fatalf("usage: %+v calls: %+v", parsed.Usage, parsed.ModelCall)
	}
	if len(parsed.ToolCall) != 1 || parsed.ToolCall[0].Status != "failed" || parsed.ToolCall[0].OutputSummary != "failed" {
		t.Fatalf("tools: %+v", parsed.ToolCall)
	}
	if parsed.Session.StartedAt.UnixMilli() != 10000 {
		t.Fatalf("inherited start time: %v", parsed.Session.StartedAt)
	}
}
