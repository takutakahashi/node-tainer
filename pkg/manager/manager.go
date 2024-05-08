package manager

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/takutakahashi/node-tainter/pkg/config"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Manager struct {
	execs  []Executor
	Node   string
	Daemon bool
	DryRun bool
	c      kubernetes.Interface
}

type Executor struct {
	Config *config.Config
	Node   string
}

func (ma Manager) Execute(ctx context.Context) error {
	if !ma.Daemon {
		return ma.ExecuteOnce(ctx)
	}
	for {
		if err := ma.ExecuteOnce(ctx); err != nil {
			logrus.Error(err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func (ma Manager) ExecuteOnce(ctx context.Context) error {
	node, err := ma.c.CoreV1().Nodes().Get(ctx, ma.Node, metav1.GetOptions{})
	if err != nil {
		return err
	}
	labels := node.GetLabels()
	taints := node.Spec.Taints
	for _, e := range ma.execs {
		if affected, err := AffectedNodeCountExceeded(
			ctx, ma.c, e.Config.MaxAffectedNodeCount, e.Config.Taints, e.Config.Labels); err != nil {
			return err
		} else if affected {
			logrus.Info("affected node count exceeded")
			continue
		}
		if err := e.ExecuteScripts(ctx); err != nil {
			taints = AddTaints(taints, e.Config.Taints)
			labels = AddLabels(labels, e.Config.Labels)
		} else {
			taints = RemoveTaints(taints, e.Config.Taints)
			labels = RemoveLabels(labels, e.Config.Labels)
		}
	}
	if ma.DryRun {
		logrus.Info("dry run")
		logrus.Info("taints:")
		for _, t := range taints {
			logrus.Infof("%s: %s", t.Key, t.Value)
		}
		logrus.Info("labels:")
		for k, v := range labels {
			logrus.Infof("%s: %s", k, v)
		}
	} else {
		node.Spec.Taints = taints
		node.SetLabels(labels)
		_, err = ma.c.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func AffectedNodeCountExceeded(ctx context.Context, c kubernetes.Interface, max int, taints []v1.Taint, labels map[string]string) (bool, error) {
	nodes, err := c.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	count := 0
	for _, node := range nodes.Items {
		if len(taints) > 0 {
			for _, t := range taints {
				if !TaintExists(node.Spec.Taints, t) {
					continue
				}
			}
		}
		if len(labels) > 0 {
			for k, v := range labels {
				if node.GetLabels()[k] != v {
					continue
				}
			}
		}
		count++
	}
	logrus.Infof("affected node count: %d", count)
	logrus.Infof("max affected node count: %d", max)
	logrus.Infof("result: %t", count > max)
	return count > max, nil

}

func TaintExists(taints []v1.Taint, t v1.Taint) bool {
	for _, tt := range taints {
		if taintEquals(tt, t) {
			return true
		}
	}
	return false
}

func taintEquals(t1, t2 corev1.Taint) bool {
	return t1.Key == t2.Key && t1.Value == t2.Value && t1.Effect == t2.Effect
}

func taintString(t corev1.Taint) string {
	return t.Key + ":" + t.Value + ":" + string(t.Effect)
}

func (m Executor) ExecuteScripts(ctx context.Context) error {
	for _, p := range m.Config.ScriptPath {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		out, err := exec.CommandContext(ctx, p).CombinedOutput()
		if os.Getenv("ENABLE_EXEC_LOG") != "" {
			logrus.Infof("%s output:", p)
			logrus.Info(string(out))
		}
		if err != nil {
			cancel()
			return err
		}
		cancel()
	}
	return nil
}

func RemoveTaints(taints []corev1.Taint, removeTaints []corev1.Taint) []corev1.Taint {
	m := map[string]corev1.Taint{}
	for _, t := range taints {
		m[taintString(t)] = t
	}
	for _, t := range removeTaints {
		delete(m, taintString(t))
	}
	ret := []corev1.Taint{}
	for _, v := range m {
		ret = append(ret, v)
	}
	return ret
}

func RemoveLabels(labels map[string]string, removeLabels map[string]string) map[string]string {
	for k := range removeLabels {
		delete(labels, k)
	}
	return labels
}

func AddTaints(taints []corev1.Taint, addTaints []corev1.Taint) []corev1.Taint {
	m := map[string]corev1.Taint{}
	for _, t := range taints {
		m[taintString(t)] = t
	}
	for _, t := range addTaints {
		m[taintString(t)] = t
	}
	ret := []corev1.Taint{}
	for _, v := range m {
		ret = append(ret, v)
	}
	return ret
}

func AddLabels(labels map[string]string, addLabels map[string]string) map[string]string {
	for k, v := range addLabels {
		labels[k] = v
	}
	return labels
}
