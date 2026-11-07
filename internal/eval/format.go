package eval

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
)

// The interchange format follows anthropics/skills skill-creator. Both criterion
// spellings are emitted because its SKILL.md and schemas.md disagree. Extensions
// survive round trips, including fields not yet understood by this runner.
func (c Corpus) MarshalJSON() ([]byte, error) {
	m := cloneFields(c.extra)
	meta := cloneFields(c.metadata)
	for k, v := range map[string]string{"source": c.Source, "commit": c.Commit, "source_sha256": c.SourceSHA256, "as_of": c.AsOf} {
		if v != "" {
			meta[k] = raw(v)
		}
	}
	name := c.SkillName
	if name == "" {
		name = "legidor"
	}
	m["skill_name"] = raw(name)
	m["evals"] = raw(c.Cases)
	m["x_legidor"] = raw(meta)
	return json.Marshal(m)
}

func (c *Corpus) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*c = Corpus{}
	_, standard := m["evals"]
	_, legacy := m["cases"]
	if standard == legacy {
		return fmt.Errorf("expected exactly one of evals or legacy cases")
	}
	if err := take(m, "skill_name", &c.SkillName); err != nil {
		return err
	}
	if standard {
		if err := take(m, "evals", &c.Cases); err != nil {
			return err
		}
		if err := take(m, "x_legidor", &c.metadata); err != nil {
			return err
		}
		for _, key := range []string{"source", "commit", "source_sha256", "as_of"} {
			if _, ok := m[key]; ok {
				return fmt.Errorf("%s must be inside x_legidor for evals format", key)
			}
		}
	} else {
		if _, ok := m["x_legidor"]; ok {
			return fmt.Errorf("legacy corpus cannot mix top-level metadata and x_legidor")
		}
		if err := take(m, "cases", &c.Cases); err != nil {
			return err
		}
		c.metadata = map[string]json.RawMessage{}
		for _, k := range []string{"source", "commit", "source_sha256", "as_of"} {
			if v, ok := m[k]; ok {
				c.metadata[k] = v
				delete(m, k)
			}
		}
	}
	for k, dest := range map[string]*string{"source": &c.Source, "commit": &c.Commit, "source_sha256": &c.SourceSHA256, "as_of": &c.AsOf} {
		if err := take(c.metadata, k, dest); err != nil {
			return err
		}
	}
	c.extra = m
	return nil
}

func (c Case) MarshalJSON() ([]byte, error) {
	m := cloneFields(c.extra)
	m["id"], m["name"], m["prompt"] = raw(c.ID), raw(c.Name), raw(c.Prompt)
	m["expected_output"] = raw(c.ExpectedOutput)
	files := c.Files
	if files == nil {
		files = []string{}
	}
	m["files"] = raw(files)
	criteria := c.Assertions
	if criteria == nil {
		criteria = []string{}
	}
	m["assertions"], m["expectations"] = raw(criteria), raw(criteria)
	meta := cloneFields(c.metadata)
	if len(c.Amendments) > 0 {
		meta["amendments"] = raw(c.Amendments)
	}
	if len(meta) > 0 {
		m["x_legidor"] = raw(meta)
	}
	return json.Marshal(m)
}

func (c *Case) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*c = Case{}
	var id *int
	if err := take(m, "id", &id); err != nil {
		return err
	}
	if id == nil {
		return fmt.Errorf("case requires an integer id")
	}
	c.ID = *id
	for key, dest := range map[string]any{"name": &c.Name, "prompt": &c.Prompt, "expected_output": &c.ExpectedOutput, "files": &c.Files, "x_legidor": &c.metadata} {
		if err := take(m, key, dest); err != nil {
			return err
		}
	}
	var assertions, expectations []string
	_, a := m["assertions"]
	_, e := m["expectations"]
	if err := take(m, "assertions", &assertions); err != nil {
		return err
	}
	if err := take(m, "expectations", &expectations); err != nil {
		return err
	}
	if a && e && !slices.Equal(assertions, expectations) {
		return fmt.Errorf("case %d: conflicting assertions and expectations", c.ID)
	}
	c.Assertions = assertions
	if e {
		c.Assertions = expectations
	}
	if _, legacy := m["amendments"]; legacy {
		if _, standard := c.metadata["amendments"]; standard {
			return fmt.Errorf("case %d: duplicate amendments metadata", c.ID)
		}
		if err := take(m, "amendments", &c.Amendments); err != nil {
			return err
		}
	} else if err := take(c.metadata, "amendments", &c.Amendments); err != nil {
		return err
	}
	if c.Name == "" {
		c.Name = fmt.Sprintf("case-%d", c.ID)
	}
	c.extra = m
	return nil
}

func cloneFields(m map[string]json.RawMessage) map[string]json.RawMessage {
	if m == nil {
		return map[string]json.RawMessage{}
	}
	return maps.Clone(m)
}

func take(m map[string]json.RawMessage, key string, dest any) error {
	b, ok := m[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	delete(m, key)
	return nil
}

func raw(v any) json.RawMessage {
	// All callers supply JSON-compatible values or previously decoded JSON.
	b, _ := json.Marshal(v)
	return b
}

// ConvertCorpus imports either format and writes a lossless interoperable file.
// The explicit research date is required for an upstream corpus without one.
// Draft scenarios without criteria and attachments are preserved, but Run rejects
// them until they can be scored and their inputs can be served safely.
func ConvertCorpus(source, dest, asOf string) error {
	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	var c Corpus
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	if asOf != "" {
		if c.AsOf != "" && c.AsOf != asOf {
			return fmt.Errorf("as_of conflicts with the existing corpus date")
		}
		c.AsOf = asOf
	}
	if err := c.validate(false); err != nil {
		return err
	}
	if c.SourceSHA256 == "" {
		c.SourceSHA256 = digest(b)
	}
	return writeJSON(dest, c)
}
