package task

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bluebrown/kobold/git"
	"github.com/bluebrown/kobold/krm"
	"github.com/bluebrown/kobold/store/model"
	"github.com/prometheus/client_golang/prometheus"
)

// the task handler is the final point of execution. After decoding, debouncing
// and aggregating the events, this handler is responsible for the actual work.
func KoboldHandler(ctx context.Context, cache string, g model.TaskGroup, runner HookRunner) ([]string, error) {
	var (
		allChanges  []krm.Change
		allWarnings []string
		lastMsg     string
		destBranch  string
	)

	if err := git.Switch(ctx, cache, g.RepoUri.Ref); err != nil {
		return nil, fmt.Errorf("git switch: %#q => %#q: %w", g.RepoUri.Repo, g.RepoUri.Ref, err)
	}

	pkgPath := filepath.Join(cache, g.RepoUri.Pkg)

	for _, ref := range g.Msgs {
		changes, warnings, err := krm.Pipeline(ctx, pkgPath, ref)
		if err != nil {
			return nil, fmt.Errorf("krm pipeline: %w", err)
		}

		allWarnings = append(allWarnings, warnings...)

		if len(changes) == 0 {
			continue
		}

		// On the first change, set up the destination branch.
		if destBranch == "" {
			if g.DestBranch.Valid {
				destBranch = g.DestBranch.String + "-" + g.Fingerprint
				if err := git.CheckoutB(ctx, cache, destBranch); err != nil {
					return nil, fmt.Errorf("git checkout -b: %w", err)
				}
			} else {
				destBranch = g.RepoUri.Ref
			}
		}

		msg, err := commitMessage(changes)
		if err != nil {
			return nil, fmt.Errorf("get commit message: %w", err)
		}

		if err := git.AddRoot(ctx, cache); err != nil {
			return nil, fmt.Errorf("git add: %w", err)
		}

		if err := git.Commit(ctx, cache, msg); err != nil {
			return nil, fmt.Errorf("git commit: %w", err)
		}

		allChanges = append(allChanges, changes...)
		lastMsg = msg
	}

	if len(allChanges) == 0 {
		return nil, nil
	}

	if err := git.Push(ctx, cache, destBranch); err != nil {
		return nil, fmt.Errorf("git push: %w", err)
	}

	metricGitPush.With(prometheus.Labels{"repo": g.RepoUri.Repo}).Inc()

	if runner == nil {
		return allWarnings, nil
	}

	if err := runner.Run(g, lastMsg, allChanges, allWarnings); err != nil {
		return allWarnings, fmt.Errorf("hook: %w", err)
	}

	return allWarnings, nil
}

func commitMessage(changes []krm.Change) (string, error) {
	msg := strings.Builder{}
	if _, err := msg.WriteString("chore(kobold): Update image refs\n"); err != nil {
		return "", fmt.Errorf("write header: %w", err)
	}

	for _, change := range changes {
		if _, err := msg.WriteString(fmt.Sprintf(" * %s: %s\n", change.Repo, change.Description)); err != nil {
			return "", fmt.Errorf("write change: %w", err)
		}
	}

	return msg.String()[:msg.Len()-1], nil
}

var _ Handler = KoboldHandler

func PrintHandler(_ context.Context, _ string, g model.TaskGroup, _ HookRunner) ([]string, error) {
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal task group: %w", err)
	}

	fmt.Println(string(b))

	return nil, nil
}

var _ Handler = PrintHandler

func ThrowHandler(_ context.Context, _ string, _ model.TaskGroup, _ HookRunner) ([]string, error) {
	return nil, fmt.Errorf("throw handler error")
}

var _ Handler = ThrowHandler
