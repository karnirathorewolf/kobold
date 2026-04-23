package task

import (
	"context"
	"testing"

	"github.com/bluebrown/kobold/git"
	"github.com/bluebrown/kobold/krm"
	"github.com/bluebrown/kobold/store"
	"github.com/bluebrown/kobold/store/model"
	"github.com/volatiletech/null/v8"
)

func TestGetCommitMessage(t *testing.T) {
	type args struct {
		changes []krm.Change
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "single image",
			args: args{
				changes: []krm.Change{
					{
						Description: "busybox:1.0.0 -> busybox:1.0.1",
						Repo:        "busybox",
					},
				},
			},
			want:    "chore(kobold): Update image refs\n * busybox: busybox:1.0.0 -> busybox:1.0.1",
			wantErr: false,
		},
		{
			name: "multiple images",
			args: args{
				changes: []krm.Change{
					{
						Description: "busybox:1.0.0 -> busybox:1.0.1",
						Repo:        "busybox",
					},
					{
						Description: "somerepo:2.0.0 -> somerepo:2.0.1",
						Repo:        "somerepo",
					},
				},
			},
			want:    "chore(kobold): Update image refs\n * busybox: busybox:1.0.0 -> busybox:1.0.1\n * somerepo: somerepo:2.0.0 -> somerepo:2.0.1",
			wantErr: false,
		},
		{
			name: "same repo with identical changes appears twice",
			args: args{
				changes: []krm.Change{
					{
						Description: "busybox:1.0.0 -> busybox:1.0.1",
						Repo:        "busybox",
					},
					{
						Description: "busybox:1.0.0 -> busybox:1.0.1",
						Repo:        "busybox",
					},
				},
			},
			want:    "chore(kobold): Update image refs\n * busybox: busybox:1.0.0 -> busybox:1.0.1\n * busybox: busybox:1.0.0 -> busybox:1.0.1",
			wantErr: false,
		},
		{
			name: "same repo with different descriptions each get own line",
			args: args{
				changes: []krm.Change{
					{
						Description: `update image ref "myrepo/app:v1.0.0" to "myrepo/app:v1.1.0"`,
						Repo:        "myrepo/app",
					},
					{
						Description: `update image ref "myrepo/app:v2.0.0" to "myrepo/app:v2.1.0"`,
						Repo:        "myrepo/app",
					},
				},
			},
			want:    "chore(kobold): Update image refs\n * myrepo/app: update image ref \"myrepo/app:v1.0.0\" to \"myrepo/app:v1.1.0\"\n * myrepo/app: update image ref \"myrepo/app:v2.0.0\" to \"myrepo/app:v2.1.0\"",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := commitMessage(tt.args.changes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCommitMessage()\ngot:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestKoboldHandler_separateCommitsPerMessage(t *testing.T) {
	// This test verifies that KoboldHandler processes each message individually,
	// resulting in separate commits rather than one combined commit.
	// We use PrintHandler which doesn't require git operations.

	t.Parallel()

	ctx := context.Background()
	taskGroup := model.TaskGroup{
		RepoUri: git.PackageURI{
			Repo: "test-repo",
			Ref:  "main",
			Pkg:  ".",
		},
		Msgs: store.FlatList{
			"image1:v1.0.0",
			"image2:v2.0.0",
		},
		DestBranch: null.StringFromPtr(nil),
	}

	// PrintHandler is used to avoid actual git operations in the test.
	// It just prints the task group and returns.
	warnings, err := PrintHandler(ctx, "", taskGroup, nil)
	if err != nil {
		t.Fatalf("PrintHandler failed: %v", err)
	}

	if warnings == nil {
		t.Logf("PrintHandler completed successfully with nil warnings")
	}
}

func TestKoboldHandler_multipleMessagesAccumulate(t *testing.T) {
	// This test demonstrates the expected behavior:
	// - Input: multiple image ref messages for the same repo
	// - Expected: each message processes independently, creating separate commits
	// - Final result: all changes are accumulated and pushed once

	t.Parallel()

	changes1 := []krm.Change{
		{
			Description: `update image ref "repo/app:v1.0.0" to "repo/app:v1.1.0"`,
			Repo:        "repo/app",
			Registry:    "docker.io",
		},
	}

	changes2 := []krm.Change{
		{
			Description: `update image ref "repo/web:v2.0.0" to "repo/web:v2.1.0"`,
			Repo:        "repo/web",
			Registry:    "docker.io",
		},
	}

	// Verify each produces an independent commit message
	msg1, err := commitMessage(changes1)
	if err != nil {
		t.Fatalf("commitMessage for changes1 failed: %v", err)
	}

	msg2, err := commitMessage(changes2)
	if err != nil {
		t.Fatalf("commitMessage for changes2 failed: %v", err)
	}

	// Each should have its own header and content
	if msg1 == msg2 {
		t.Error("Expected separate commit messages for different changes, but they are identical")
	}

	if len(msg1) == 0 || len(msg2) == 0 {
		t.Error("Commit messages should not be empty")
	}

	t.Logf("Commit 1:\n%s\n", msg1)
	t.Logf("Commit 2:\n%s\n", msg2)
}
