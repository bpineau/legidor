package eval

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAnthropicCorpusRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evals.json")
	input := `{"skill_name":"synthetic","custom":{"keep":true},"x_legidor":{"as_of":"2026-11-07","source":"public-fixture","future":42},"evals":[{"id":0,"prompt":"Calculate 2 + 2","expected_output":"Four","files":["evals/files/input.txt"],"expectations":["Answers four"],"custom_case":true,"x_legidor":{"amendments":["Clarified question"],"future":true}}]}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCorpus(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Cases[0].Name != "case-0" || c.Cases[0].Assertions[0] != "Answers four" {
		t.Fatalf("lost scenario: %+v", c)
	}
	if err := writeJSON(path, c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["cases"] != nil || doc["skill_name"] != "synthetic" || doc["custom"] == nil {
		t.Fatalf("not interoperable: %s", data)
	}
	item := doc["evals"].([]any)[0].(map[string]any)
	if item["expected_output"] != "Four" || item["files"].([]any)[0] != "evals/files/input.txt" || item["custom_case"] != true || item["x_legidor"].(map[string]any)["future"] != true || doc["x_legidor"].(map[string]any)["future"] != float64(42) {
		t.Fatalf("lost data: %s", data)
	}
	if _, err := LoadCorpus(path); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusAliasesAndAmbiguity(t *testing.T) {
	for _, criteria := range []string{`"assertions":["a"]`, `"expectations":["a"]`, `"assertions":["a"],"expectations":["a"]`} {
		var c Corpus
		err := json.Unmarshal([]byte(`{"skill_name":"test","evals":[{"id":1,"prompt":"p",`+criteria+`}]}`), &c)
		if err != nil || len(c.Cases) != 1 || len(c.Cases[0].Assertions) != 1 {
			t.Fatalf("%s: %+v %v", criteria, c, err)
		}
	}
	for _, input := range []string{
		`{"evals":[],"cases":[]}`,
		`{"evals":[{"id":1,"prompt":"p","assertions":["a"],"expectations":["b"]}]}`,
		`{"evals":[{"prompt":"missing ID","assertions":["a"]}]}`,
	} {
		var c Corpus
		if err := json.Unmarshal([]byte(input), &c); err == nil {
			t.Fatalf("accepted ambiguous corpus: %s", input)
		}
	}
}

func TestLegacyCorpusPreservesMetadata(t *testing.T) {
	var c Corpus
	legacy := `{"source":"upstream","commit":"abc","source_sha256":"123","as_of":"2026-11-07","cases":[{"id":1,"name":"old","prompt":"p","assertions":["a"],"amendments":["clarification"]}]}`
	if err := json.Unmarshal([]byte(legacy), &c); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var restored Corpus
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Source != "upstream" || restored.Commit != "abc" || restored.SourceSHA256 != "123" || restored.AsOf != c.AsOf || restored.Cases[0].Amendments[0] != "clarification" {
		t.Fatalf("lost provenance: %s", b)
	}
}

func TestRunnerRejectsUnservedAttachmentsBeforeInvocation(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "corpus.json")
	input := `{"skill_name":"test","x_legidor":{"as_of":"2026-11-07"},"evals":[{"id":1,"prompt":"Read attached file","files":["input.pdf"],"expectations":["a"]}]}`
	os.WriteFile(p, []byte(input), 0600)
	err := Run(context.Background(), Options{Corpus: p, Output: filepath.Join(root, "out"), Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test", CacheMode: "cold", Repetitions: 1, Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "attachments") {
		t.Fatalf("must reject unsupported attachments explicitly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "out")); !os.IsNotExist(err) {
		t.Fatal("started a run with unavailable inputs")
	}
}

func TestConvertCorpusRequiresAndPreservesResearchDate(t *testing.T) {
	root := t.TempDir()
	source, out := filepath.Join(root, "source.json"), filepath.Join(root, "output.json")
	os.WriteFile(source, []byte(`{"skill_name":"draft","evals":[{"id":0,"prompt":"p","expected_output":"answer","files":[]}]}`), 0600)
	if err := ConvertCorpus(source, out, ""); err == nil {
		t.Fatal("silently assumed a date")
	}
	if err := ConvertCorpus(source, out, "2026-02-30"); err == nil {
		t.Fatal("accepted invalid date")
	}
	if err := ConvertCorpus(source, out, "2026-11-07"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCorpus(out); err == nil {
		t.Fatal("allowed a scored run without criteria")
	}
	if err := ConvertCorpus(out, source, "2027-01-01"); err == nil {
		t.Fatal("silently rewrote applicability date")
	}
}

func TestImportAllPaperassePreservesUpstreamCases(t *testing.T) {
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, "fiscaliste", "evals"), 0700)
	input := `{"skill_name":"fiscaliste","evals":[{"id":1,"name":"example","prompt":"p","expected_output":"original answer","assertions":["a"],"files":[]},{"id":42,"prompt":"second","expected_output":"other answer","expectations":["b"],"files":[]}]}`
	os.WriteFile(filepath.Join(repo, "fiscaliste", "evals", "evals.json"), []byte(input), 0600)
	os.WriteFile(filepath.Join(repo, "LICENSE"), []byte("Synthetic fixture license\n"), 0600)
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "Synthetic fixtures"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", b, err)
		}
	}
	out := t.TempDir()
	if err := ImportPaperasseCorpus(repo, out, true, "2026-11-07"); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCorpus(filepath.Join(out, "all.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Cases) != 2 || c.Cases[1].ID != 42 || c.Cases[0].ExpectedOutput != "original answer" || len(c.Cases[0].Amendments) != 0 || c.SourceSHA256 != digest([]byte(input)) || c.Commit == "" {
		t.Fatalf("changed unamended source: %+v", c)
	}
	if string(mustRead(t, filepath.Join(out, "PAPERASSE-LICENSE"))) != "Synthetic fixture license\n" {
		t.Fatal("lost license")
	}
	// A future upstream extension must not disappear or silently change date.
	extended := strings.Replace(input, `"skill_name":"fiscaliste"`, `"skill_name":"fiscaliste","x_legidor":{"as_of":"2026-10-01","source":"original-source","source_sha256":"original-hash","custom":{"keep":true}}`, 1)
	os.WriteFile(filepath.Join(repo, "fiscaliste", "evals", "evals.json"), []byte(extended), 0600)
	if err := ImportPaperasseCorpus(repo, out, true, "2026-11-07"); err == nil {
		t.Fatal("silently changed upstream date")
	}
	if err := ImportPaperasseCorpus(repo, out, true, "2026-10-01"); err != nil {
		t.Fatal(err)
	}
	c, err = LoadCorpus(filepath.Join(out, "all.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Source != "original-source" || c.SourceSHA256 != "original-hash" || c.AsOf != "2026-10-01" || c.metadata["custom"] == nil || c.metadata["imports"] == nil {
		t.Fatalf("lost upstream metadata: %+v", c)
	}
}

func TestRunnerRejectsUnknownSelectedCase(t *testing.T) {
	calls := fakeCLI(t)
	root := t.TempDir()
	p := filepath.Join(root, "corpus.json")
	writeJSON(p, Corpus{AsOf: "2026-11-07", Cases: []Case{{ID: 1, Name: "one", Prompt: "p", Assertions: []string{"a"}}}})
	err := Run(context.Background(), Options{Corpus: p, Output: filepath.Join(root, "out"), IDs: []string{"1", "42"}, Agents: []string{"codex"}, Arms: []string{"none"}, CodexModel: "test", CacheMode: "cold", Repetitions: 1, Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "case ID") {
		t.Fatalf("silently ran a subset: %v", err)
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatal("called model for invalid selection")
	}
}
