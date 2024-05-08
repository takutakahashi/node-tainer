package manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/takutakahashi/node-tainter/pkg/config"
	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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

func TestManager_ExecuteOnce(t *testing.T) {
	taints := []corev1.Taint{{Key: "key1", Value: "value1"}}
	labels := map[string]string{"key1": "value1"}
	node := newNode("test-node", labels, taints)

	fakeClientset := fake.NewSimpleClientset(node)

	ma := Manager{
		execs:  []Executor{{Config: &config.Config{}}},
		Node:   "test-node",
		Daemon: false,
		DryRun: false,
		c:      fakeClientset,
	}

	err := ma.ExecuteOnce(context.Background())

	assert.NoError(t, err)
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
