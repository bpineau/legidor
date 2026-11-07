package eval

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Measurement normalizes provider usage without equating cache reads with free tokens.
// InputTokens includes cache reads and cache writes for both providers.
type Measurement struct {
	Answer           string   `json:"answer"`
	Model            string   `json:"model,omitempty"`
	InputTokens      int64    `json:"input_tokens"`
	OutputTokens     int64    `json:"output_tokens"`
	CacheReadTokens  int64    `json:"cache_read_tokens"`
	CacheWriteTokens int64    `json:"cache_write_tokens"`
	ToolCalls        int      `json:"tool_calls"`
	CostUSD          *float64 `json:"estimated_cost_usd"`
	UsageComplete    bool     `json:"usage_complete"`
}

// ParseTrace requires an explicit successful terminal event and a final answer.
// Partial usage is returned alongside an error so failed attempts remain observable.
func ParseTrace(agent string, r io.Reader) (Measurement, error) {
	var m Measurement
	done := false
	var failure error
	partial := map[string]Measurement{}
	type usage struct {
		Input      int64 `json:"input_tokens"`
		Output     int64 `json:"output_tokens"`
		Cached     int64 `json:"cached_input_tokens"`
		Read       int64 `json:"cache_read_input_tokens"`
		Write      int64 `json:"cache_creation_input_tokens"`
		CodexWrite int64 `json:"cache_write_input_tokens"`
	}
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 4096), 16*1024*1024)
	for scan.Scan() {
		var e struct {
			Type    string   `json:"type"`
			Subtype string   `json:"subtype"`
			Model   string   `json:"model"`
			Result  string   `json:"result"`
			IsError bool     `json:"is_error"`
			Cost    *float64 `json:"total_cost_usd"`
			Usage   *usage   `json:"usage"`
			Item    struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
			Message struct {
				ID      string `json:"id"`
				Usage   *usage `json:"usage"`
				Content []struct {
					Type string `json:"type"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(scan.Bytes(), &e); err != nil {
			return m, fmt.Errorf("invalid trace: %w", err)
		}
		if agent == "claude" {
			if e.Type == "system" && e.Subtype == "init" {
				m.Model = e.Model
			}
			if e.Type == "assistant" {
				if e.Message.ID != "" && e.Message.Usage != nil && !done {
					old := partial[e.Message.ID]
					u := e.Message.Usage
					next := Measurement{InputTokens: max(old.InputTokens, u.Input+u.Read+u.Write), OutputTokens: max(old.OutputTokens, u.Output), CacheReadTokens: max(old.CacheReadTokens, u.Read), CacheWriteTokens: max(old.CacheWriteTokens, u.Write)}
					m.InputTokens += next.InputTokens - old.InputTokens
					m.OutputTokens += next.OutputTokens - old.OutputTokens
					m.CacheReadTokens += next.CacheReadTokens - old.CacheReadTokens
					m.CacheWriteTokens += next.CacheWriteTokens - old.CacheWriteTokens
					partial[e.Message.ID] = next
				}
				for _, c := range e.Message.Content {
					if c.Type == "tool_use" {
						m.ToolCalls++
					}
				}
			}
			if e.Type == "result" {
				done = true
				m.Answer = e.Result
				if e.Usage != nil {
					m.UsageComplete = true
					m.InputTokens = e.Usage.Input + e.Usage.Read + e.Usage.Write
					m.OutputTokens = e.Usage.Output
					m.CacheReadTokens = e.Usage.Read
					m.CacheWriteTokens = e.Usage.Write
				}
				m.CostUSD = e.Cost
				if e.IsError || e.Subtype != "success" {
					failure = fmt.Errorf("Claude terminal status: %s", e.Subtype)
				}
			}
		} else if agent == "codex" {
			if e.Type == "item.completed" {
				switch e.Item.Type {
				case "agent_message":
					m.Answer = e.Item.Text
				case "web_search", "mcp_tool_call", "command_execution":
					m.ToolCalls++
				}
			}
			if e.Type == "turn.completed" {
				done = true
				if e.Usage != nil {
					m.UsageComplete = true
					m.InputTokens = e.Usage.Input
					m.OutputTokens = e.Usage.Output
					m.CacheReadTokens = e.Usage.Cached
					m.CacheWriteTokens = e.Usage.CodexWrite
				}
			}
			if e.Type == "turn.failed" || e.Type == "error" {
				failure = errors.New("Codex failed; inspect trace")
			}
		} else {
			return m, fmt.Errorf("unknown agent %q", agent)
		}
	}
	if err := scan.Err(); err != nil {
		return m, err
	}
	if failure != nil {
		return m, failure
	}
	if !done || m.Answer == "" {
		return m, errors.New("missing successful completion or final answer")
	}
	return m, nil
}
