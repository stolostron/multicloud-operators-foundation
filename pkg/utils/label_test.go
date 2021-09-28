package utils

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCloneAndAddLabel(t *testing.T) {
	type args struct {
		labels     map[string]string
		labelKey   string
		labelValue string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{"case1:", args{labels: map[string]string{"label1": "va1", "label2": "va2"}, labelKey: "key", labelValue: "value"},
			map[string]string{"label1": "va1", "label2": "va2", "key": "value"}},
		{"case2:", args{labels: nil, labelKey: "", labelValue: ""}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CloneAndAddLabel(tt.args.labels, tt.args.labelKey, tt.args.labelValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneAndAddLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchLabelForLabelSelector(t *testing.T) {
	type args struct {
		targetLabels  map[string]string
		labelSelector *metav1.LabelSelector
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1:", args{targetLabels: map[string]string{"label1": "va1", "label2": "va2"}}, true},
		{"case2:", args{targetLabels: map[string]string{"label1": "va1", "label2": "va2"},
			labelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "va1", "label2": "va2"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchLabelForLabelSelector(tt.args.targetLabels, tt.args.labelSelector); got != tt.want {
				t.Errorf("MatchLabelForLabelSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeMap(t *testing.T) {
	type args struct {
		modified bool
		existing *map[string]string
		required map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1", args{modified: false, required: map[string]string{"label1": "va1"}, existing: &map[string]string{"label1": "va1", "label2": "va2"}}, false},
		{"case2", args{modified: false, required: map[string]string{"label1": "va1"}, existing: &map[string]string{}}, true},
		{"case3", args{modified: false, required: map[string]string{"label1-": "va1"}, existing: &map[string]string{"label1": "va1", "label2": "va2"}}, true},
		{"case4", args{modified: false, required: map[string]string{"label1": "va1"}, existing: &map[string]string{"label1": "va1", "label2": "va2"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeMap(&tt.args.modified, tt.args.existing, tt.args.required)
			if tt.want != tt.args.modified {
				t.Errorf("failed to merge map")
			}
		})
	}
}
func TestStringToMap(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		rst  map[string]string
	}{
		{
			name: "case1",
			str:  "",
			rst:  nil,
		},
		{
			name: "case2",
			str:  "app=opt,zone=east-1",
			rst: map[string]string{
				"app":  "opt",
				"zone": "east-1",
			},
		},
	}

	for _, testCase := range testCases {
		rst := StringToMap(testCase.str)
		if len(rst) != len(testCase.rst) {
			t.Errorf("test case %s fail", testCase.name)
		}
	}
}

func TestStrings(t *testing.T) {
	str := []string{"label1", "label2", "label3"}
	outStr := RemoveString(str, "label2")
	if ContainsString(outStr, "label2") {
		t.Errorf("failed to remove string from slice")
	}
}

func TestSyncMapField(t *testing.T) {
	var modified = false
	type args struct {
		modified  *bool
		existing  map[string]string
		required  map[string]string
		syncFiled string
	}
	tests := []struct {
		name         string
		args         args
		wantModified bool
		wantExisting map[string]string
	}{
		{
			name: "label in existing but not in required, should delete it",
			args: args{
				modified:  &modified,
				required:  map[string]string{"label1": "va1"},
				existing:  map[string]string{"sync": "va1", "label2": "va2"},
				syncFiled: "sync",
			},
			wantModified: true,
			wantExisting: map[string]string{"label2": "va2"},
		},
		{
			name: "label in existing and in required, should sync it",
			args: args{
				modified:  &modified,
				required:  map[string]string{"sync": "va1", "label2": "va2"},
				existing:  map[string]string{"sync": "va2", "label1": "va1"},
				syncFiled: "sync",
			},
			wantModified: true,
			wantExisting: map[string]string{"sync": "va1", "label1": "va1"},
		},
		{
			name: "label in existing same as in required, should not sync it",
			args: args{
				modified:  &modified,
				required:  map[string]string{"sync": "va1", "label2": "va2"},
				existing:  map[string]string{"sync": "va1", "label1": "va1"},
				syncFiled: "sync",
			},
			wantModified: false,
			wantExisting: map[string]string{"sync": "va1", "label1": "va1"},
		},
		{
			name: "label not in existing and in required, should sync it",
			args: args{
				modified:  &modified,
				required:  map[string]string{"sync": "va1", "label2": "va2"},
				existing:  map[string]string{"label1": "va1"},
				syncFiled: "sync",
			},
			wantModified: true,
			wantExisting: map[string]string{"sync": "va1", "label1": "va1"},
		},
		{
			name: "required is nil map, sync label do not in existing",
			args: args{
				modified:  &modified,
				required:  nil,
				existing:  map[string]string{"label1": "va1"},
				syncFiled: "sync",
			},
			wantModified: false,
			wantExisting: map[string]string{"label1": "va1"},
		},
		{
			name: "required is nil map, sync label in existing",
			args: args{
				modified:  &modified,
				required:  nil,
				existing:  map[string]string{"sync": "va1"},
				syncFiled: "sync",
			},
			wantModified: true,
			wantExisting: map[string]string{},
		},
		{
			name: "existing is nil map, sync label in required",
			args: args{
				modified:  &modified,
				required:  map[string]string{"sync": "va1"},
				existing:  nil,
				syncFiled: "sync",
			},
			wantModified: true,
			wantExisting: map[string]string{"sync": "va1"},
		},
		{
			name: "existing is nil map, sync label not in required",
			args: args{
				modified:  &modified,
				required:  map[string]string{"label1": "va1"},
				existing:  nil,
				syncFiled: "sync",
			},
			wantModified: false,
			wantExisting: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SyncMapField(tt.args.modified, &tt.args.existing, tt.args.required, tt.args.syncFiled)
			if tt.wantModified != *tt.args.modified {
				t.Errorf("case: %v, failed to sync map, want wantModified:%v, actual:%v", tt.name, tt.wantModified, *tt.args.modified)
			}
			if !reflect.DeepEqual(tt.wantExisting, tt.args.existing) {
				t.Errorf("case: %v, failed to sync map, want wantExisting:%v, actualExisting:%v", tt.name, tt.wantExisting, tt.args.existing)
			}
		})
	}
}
