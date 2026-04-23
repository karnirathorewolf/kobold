package task

import (
	"testing"

	"github.com/bluebrown/kobold/krm"
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
