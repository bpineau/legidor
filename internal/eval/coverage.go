package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// unstartedSamples reconstructs the matrix without calling agents. Old derived
// manifests may lack a plan; report that coverage as unknown instead of inventing
// an apparently complete campaign from the subset of samples that survived.
func unstartedSamples(root string, cases map[int]int, samples map[string]Sample) ([]Sample, bool, error) {
	b, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var m struct {
		Agents       []string `json:"agents"`
		Arms         []string `json:"arms"`
		IDs          []string `json:"ids"`
		Repetitions  int      `json:"repetitions"`
		ClaudeModel  string   `json:"claude_model"`
		CodexModel   string   `json:"codex_model"`
		ClaudeModels []string `json:"claude_models"`
		CodexModels  []string `json:"codex_models"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false, err
	}
	if len(m.Agents) == 0 || len(m.Arms) == 0 || m.Repetitions < 1 || len(cases) == 0 {
		return nil, false, nil
	}
	type cell struct {
		agent, model, arm string
		id, rep           int
	}
	seen := map[cell]bool{}
	// An unstarted sample has no observed model version. Reuse the observed
	// version only when it is unambiguous for that requested agent/model.
	observed := map[string]string{}
	ambiguous := map[string]bool{}
	for _, s := range samples {
		requested := s.RequestedModel
		if requested == "" {
			requested = s.Model
		}
		seen[cell{s.Agent, requested, s.Arm, s.CaseID, s.Repetition}] = true
		key := s.Agent + "/" + requested
		if old := observed[key]; old != "" && old != s.Model {
			ambiguous[key] = true
		}
		observed[key] = s.Model
	}
	var missing []Sample
	for _, agent := range m.Agents {
		models, fallback := m.ClaudeModels, m.ClaudeModel
		if agent == "codex" {
			models, fallback = m.CodexModels, m.CodexModel
		}
		if len(models) == 0 {
			models = []string{fallback}
		}
		for _, model := range models {
			if model == "" {
				return nil, false, nil
			}
			actual := model
			if key := agent + "/" + model; observed[key] != "" && !ambiguous[key] {
				actual = observed[key]
			}
			for _, arm := range m.Arms {
				for id := range cases {
					if len(m.IDs) > 0 && !contains(m.IDs, strconv.Itoa(id)) {
						continue
					}
					for rep := 1; rep <= m.Repetitions; rep++ {
						if !seen[cell{agent, model, arm, id, rep}] {
							missing = append(missing, Sample{Agent: agent, RequestedModel: model, Arm: arm, CaseID: id, Repetition: rep, Measurement: Measurement{Model: actual}})
						}
					}
				}
			}
		}
	}
	return missing, true, nil
}
