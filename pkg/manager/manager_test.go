package manager

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/takutakahashi/node-tainter/pkg/config"
	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	taints               = []corev1.Taint{{Key: "key1", Value: "value1"}}
	labels               = map[string]string{"key1": "value1"}
	noTaints             = []corev1.Taint{}
	noLabels             = map[string]string{}
	ThreeNodesWithLabels = []*corev1.Node{
		newNode("test-node-1", labels, noTaints),
		newNode("test-node-2", labels, noTaints),
		newNode("test-node-3", labels, noTaints),
	}
	ThreeNodesWithTaints = []*corev1.Node{
		newNode("test-node-1", noLabels, taints),
		newNode("test-node-2", noLabels, taints),
		newNode("test-node-3", noLabels, taints),
	}
	ThreeNodesWithLabelsAndTaints = []*corev1.Node{
		newNode("test-node-1", labels, taints),
		newNode("test-node-2", labels, taints),
		newNode("test-node-3", labels, taints),
	}
	ThreeNodesWithNoLabelsAndNoTaints = []*corev1.Node{
		newNode("test-node-1", noLabels, noTaints),
		newNode("test-node-2", noLabels, noTaints),
		newNode("test-node-3", noLabels, noTaints),
	}
)

func newNode(name string, labels map[string]string, taints []corev1.Taint) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.NodeSpec{
			Taints: taints,
		},
	}
}

func TestExecutor_ExecuteScripts(t *testing.T) {
	ex := Executor{
		Config: &config.Config{
			ScriptPath: []string{"../../misc/scripts/ok.sh"},
		},
		Node: "test-node",
	}
	err := ex.ExecuteScripts(context.Background())

	assert.NoError(t, err)
}

func TestAffectedNodeCountExceeded(t *testing.T) {
	testCases := []struct {
		name             string
		existingNodes    []corev1.Node
		maxAffectedNodes int
		inputTaints      []corev1.Taint
		inputLabels      map[string]string
		wantExceeded     bool
		wantErr          bool
	}{
		{
			name:             "no nodes with taints or labels",
			existingNodes:    []corev1.Node{{}},
			maxAffectedNodes: 1,
			inputTaints:      []corev1.Taint{{Key: "key1"}},
			inputLabels:      map[string]string{"app": "test-app"},
			wantExceeded:     false,
		},
		{
			name: "toleration and label match, one node, not exceeded",
			existingNodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "key1"}}},
				},
			},
			maxAffectedNodes: 2,
			inputTaints:      []corev1.Taint{{Key: "key1"}},
			inputLabels:      map[string]string{"app": "test-app"},
			wantExceeded:     false,
		},
		{
			name: "toleration and label match, three node, exceeded with LABEL",
			existingNodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{}},
				},
			},
			maxAffectedNodes: 2,
			inputTaints:      []corev1.Taint{},
			inputLabels:      map[string]string{"app": "test-app"},
			wantExceeded:     true,
		},
		{
			name: "toleration and label match, three node, exceeded with TAINTS",
			existingNodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "key1"}}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "key1"}}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}},
					Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "key1"}}},
				},
			},
			maxAffectedNodes: 2,
			inputTaints:      []corev1.Taint{{Key: "key1"}},
			inputLabels:      map[string]string{},
			wantExceeded:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			for i, node := range tc.existingNodes {
				node.Name = fmt.Sprintf("test-node-%d", i)
				client.CoreV1().Nodes().Create(context.Background(), &node, metav1.CreateOptions{})
			}

			exceeded, err := AffectedNodeCountExceeded(context.Background(), client, tc.maxAffectedNodes, tc.inputTaints, tc.inputLabels)

			assert.Equal(t, tc.wantExceeded, exceeded)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaintExists(t *testing.T) {
	type args struct {
		taints []v1.Taint
		t      v1.Taint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "taint exists",
			args: args{
				taints: []v1.Taint{
					{
						Key: "key1",
					},
				},
				t: v1.Taint{
					Key: "key1",
				},
			},
			want: true,
		},
		{
			name: "taint does not exist",
			args: args{
				taints: []v1.Taint{
					{
						Key: "key1",
					},
				},
				t: v1.Taint{
					Key: "key2",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TaintExists(tt.args.taints, tt.args.t); got != tt.want {
				t.Errorf("TaintExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestRemoveLabels(t *testing.T) {
	type args struct {
		labels       map[string]string
		removeLabels map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "remove one label",
			args: args{
				labels:       map[string]string{"key1": "value1", "key2": "value2"},
				removeLabels: map[string]string{"key1": "value1"},
			},
			want: map[string]string{"key2": "value2"},
		},
		{
			name: "remove two labels",
			args: args{
				labels:       map[string]string{"key1": "value1", "key2": "value2"},
				removeLabels: map[string]string{"key1": "value1", "key2": "value2"},
			},
			want: map[string]string{},
		},
		{
			name: "remove no labels",
			args: args{
				labels:       map[string]string{"key1": "value1", "key2": "value2"},
				removeLabels: map[string]string{},
			},
			want: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "remove non-existing label",
			args: args{
				labels:       map[string]string{"key1": "value1", "key2": "value2"},
				removeLabels: map[string]string{"key3": "value3"},
			},
			want: map[string]string{"key1": "value1", "key2": "value2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveLabels(tt.args.labels, tt.args.removeLabels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddTaints(t *testing.T) {
	type args struct {
		taints    []corev1.Taint
		addTaints []corev1.Taint
	}
	tests := []struct {
		name string
		args args
		want []corev1.Taint
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddTaints(tt.args.taints, tt.args.addTaints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddTaints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddLabels(t *testing.T) {
	type args struct {
		labels    map[string]string
		addLabels map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddLabels(tt.args.labels, tt.args.addLabels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	type args struct {
		configs []*config.Config
		node    string
		daemon  bool
		dryrun  bool
		c       kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
		want Manager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewManager(tt.args.configs, tt.args.node, tt.args.daemon, tt.args.dryrun, tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Execute(t *testing.T) {
	type fields struct {
		execs  []Executor
		Node   string
		Daemon bool
		DryRun bool
		c      kubernetes.Interface
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ma := Manager{
				execs:  tt.fields.execs,
				Node:   tt.fields.Node,
				Daemon: tt.fields.Daemon,
				DryRun: tt.fields.DryRun,
				c:      tt.fields.c,
			}
			if err := ma.Execute(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Manager.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_ExecuteOnce(t *testing.T) {
	type fields struct {
		execs  []Executor
		Node   string
		Daemon bool
		DryRun bool
		c      kubernetes.Interface
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ma := Manager{
				execs:  tt.fields.execs,
				Node:   tt.fields.Node,
				Daemon: tt.fields.Daemon,
				DryRun: tt.fields.DryRun,
				c:      tt.fields.c,
			}
			if err := ma.ExecuteOnce(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Manager.ExecuteOnce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_taintEquals(t *testing.T) {
	type args struct {
		t1 corev1.Taint
		t2 corev1.Taint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taintEquals(tt.args.t1, tt.args.t2); got != tt.want {
				t.Errorf("taintEquals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_taintString(t *testing.T) {
	type args struct {
		t corev1.Taint
	}
	tests := []struct {
		name string
		args args
		want string
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taintString(tt.args.t); got != tt.want {
				t.Errorf("taintString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveTaints(t *testing.T) {
	type args struct {
		taints       []corev1.Taint
		removeTaints []corev1.Taint
	}
	tests := []struct {
		name string
		args args
		want []corev1.Taint
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveTaints(tt.args.taints, tt.args.removeTaints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveTaints() = %v, want %v", got, tt.want)
			}
		})
	}
}
