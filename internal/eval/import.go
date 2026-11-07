package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ImportPaperasse imports the public pilot only, never upstream executable code.
// Explicit amendments make the synthetic facts and scoring dates unambiguous.
func ImportPaperasse(repo, dest string) error {
	p := filepath.Join(repo, "fiscaliste", "evals", "evals.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var upstream struct {
		Evals []Case `json:"evals"`
	}
	if err = json.Unmarshal(data, &upstream); err != nil {
		return err
	}
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	corpus := Corpus{Source: "https://github.com/romainsimon/paperasse/blob/" + strings.TrimSpace(string(commit)) + "/fiscaliste/evals/evals.json", Commit: strings.TrimSpace(string(commit)), SourceSHA256: digest(data), AsOf: "2026-11-07"}
	additions := map[int]string{
		1:  "",
		3:  "Cas synthétique précisé : célibataire, une part, versement et revenus étudiés en 2025. Le salaire imposable 1AJ était également de 80 000 EUR en 2024. Aucun autre revenu, plafond reportable, mutualisation, frais réels, cotisation professionnelle de retraite supplémentaire ou abondement à déduire du plafond. L'avis indique un plafond disponible de 7 200 EUR.",
		4:  "Revenus de 2025, déclaration 2026, une part, dividendes éligibles à l'abattement, prélèvements sociaux prélevés à la source en 2025 par un établissement payeur français. Aucun autre revenu, charge, réduction ou crédit. Comparer les impôts totaux, décote et CSG déductible incluses.",
		6:  "Cas synthétique : acquisition définitive en 2025 d'actions gratuites dans un plan français qualifiant autorisé en 2020. Célibataire, une part, salaire net imposable 1AJ 60 000 EUR. Aucune action vendue en 2025. Expliquer également la fiscalité conditionnelle si ces actions avaient été vendues en 2025 pour leur valeur d'acquisition, sans autre gain. Ne pas présumer que le quotient est automatiquement acquis. Pour une future plus-value de cession, prendre une cession en 2026.",
		11: "Hors mécanisme de lissage : les revenus des deux années précédentes ne permettent pas son application. La question porte uniquement sur la CEHR.",
		12: "Revenus de 2025. Travaux déductibles ordinaires, hors dispositif temporaire de rénovation énergétique et hors dispositif spécial. Aucun déficit antérieur.",
		13: "Retrait total en 2026. Pour ce calcul synthétique, les 10 000 EUR de gains ont tous été acquis en 2026. Aucune exception liée à la situation personnelle n'est applicable ; les présenter seulement comme réserves générales.",
		14: "Rachat en 2026. Tous les versements ont été effectués en 2016, contrat en unités de compte, aucun prélèvement social déjà payé sur les gains, aucun autre rachat cette année, abattement annuel intégralement disponible.",
	}
	for _, c := range upstream.Evals {
		addition, ok := additions[c.ID]
		if !ok {
			continue
		}
		c.Prompt = ASCII(c.Prompt)
		for i := range c.Assertions {
			c.Assertions[i] = ASCII(c.Assertions[i])
		}
		c.Amendments = []string{"Normalize typographic quotes and dashes to ASCII; preserve upstream amounts and scoring intent."}
		if addition != "" {
			c.Prompt += "\n\n" + addition
			c.Amendments = append(c.Amendments, "Append explicit synthetic facts and dates; see the prompt's final paragraph.")
		}
		if c.ID == 3 {
			c.Assertions[0] = "L'économie immédiate est calculée sur le montant déductible : 7 200 EUR x 30 % = 2 160 EUR, et non 8 000 EUR x 30 %."
			c.Assertions[3] = "Le plafond disponible de 7 200 EUR est respecté (référence aux revenus professionnels de l'année précédente), donc 800 EUR du versement ne sont pas déductibles."
			c.Amendments = append(c.Amendments, "Clarify the available prior-year ceiling and deductible amount in criteria 0 and 3.")
		}
		if c.ID == 6 {
			c.Assertions[0] = "Aucun impôt sur le gain d'acquisition du plan qualifiant au seul vesting en 2025, sans cession ; la cession déclenche l'imposition."
			c.Assertions[3] = "La réponse ne confond pas le régime du plan qualifiant précisé avec celui d'un salaire ordinaire ou de la fraction de gain excédant 300 000 EUR. Elle n'a pas à développer un plan non qualifiant hors scénario."
			c.Amendments = append(c.Amendments, "Criteria 0 and 3 follow the explicitly qualified plan; do not require an irrelevant non-qualified-plan digression.")
		}
		corpus.Cases = append(corpus.Cases, c)
	}
	if len(corpus.Cases) != 8 {
		return fmt.Errorf("expected eight pilot cases, got %d", len(corpus.Cases))
	}
	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	license, err := os.ReadFile(filepath.Join(repo, "LICENSE"))
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dest, "PAPERASSE-LICENSE"), license, 0644); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dest, "pilot.json"), corpus)
}
