package eval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Campaign executes the matrix and then grades it with one explicit judge.
// No retries are automatic. Quota errors and interrupted runs retain their
// evidence and stop before grading, so unattended campaigns have a fixed scope.
func Campaign(ctx context.Context, o Options, judgeAgent, judgeModel string, gradingTimeout time.Duration) error {
	if (judgeAgent != "claude" && judgeAgent != "codex") || judgeModel == "" || gradingTimeout <= 0 {
		return fmt.Errorf("judge agent, explicit judge model and positive grading timeout are required")
	}
	if err := o.validate(); err != nil {
		return err
	}
	root, err := filepath.Abs(o.Output)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(root, ".campaign.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("campaign already running or lock unavailable: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	started := time.Now().UTC()
	status := func(phase string, failure error) error {
		state := map[string]any{"phase": phase, "pid": os.Getpid(), "started_at": started, "updated_at": time.Now().UTC(), "judge_agent": judgeAgent, "judge_model": judgeModel}
		if failure != nil {
			state["error"] = failure.Error()
		}
		return writeJSON(filepath.Join(root, "campaign.json"), state)
	}
	if err = status("running", nil); err != nil {
		return err
	}
	if err = Run(ctx, o); err != nil {
		_ = status("execution_failed", err)
		return err
	}
	if err = status("grading", nil); err != nil {
		return err
	}
	// Use the immutable evaluated snapshot, never a subsequently edited corpus.
	if err = GradeAnswers(ctx, root, filepath.Join(root, "corpus.json"), judgeAgent, judgeModel, gradingTimeout); err != nil {
		_ = status("grading_failed", err)
		return err
	}
	return status("complete", nil)
}
