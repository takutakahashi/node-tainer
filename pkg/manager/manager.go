package manager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/slkm"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Manager struct {
	ScriptPath          []string
	Taint               string
	Node                string
	Daemon              bool
	DryRun              bool
	SlackWebhook        string
	SlackChannel        string
	MaxTaintedNodeCount int
	c                   *kubernetes.Clientset
}

func (m Manager) Execute() error {
	if !m.Daemon {
		return m.ExecuteOnce()
	}
	for {
		if err := m.ExecuteOnce(); err != nil {
			logrus.Error(err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func (m Manager) ExecuteOnce() error {
	err := m.ExecuteScripts()
	if err == nil {
		return m.RemoveTaint()
	}
	logrus.Info(err)
	logrus.Info("start taint")
	return m.AddTaint()
}

func (m Manager) NotifySlack(message string) error {
	if m.SlackWebhook == "" || m.SlackChannel == "" {
		return nil
	}
	ctx := context.Background()
	c, err := slkm.New()
	if err != nil {
		return err
	}
	if m.DryRun {
		c.SetUsername("[DRY-RUN] node-tainter")

	} else {
		c.SetUsername("node-tainter")
	}
	c.SetWebhookURL(m.SlackWebhook)
	blocks := []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", message, false, false), nil, nil),
	}
	if err := c.PostMessage(ctx, m.SlackChannel, blocks...); err != nil {
		return err
	}
	return nil
}

func (m Manager) ExecuteScripts() error {
	for _, p := range m.ScriptPath {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func (m Manager) RemoveTaint() error {
	if m.DryRun {
		logrus.Infof("dryrun: remove taint, %s, %s", m.Node, m.Taint)
		return nil
	}
	if m.c == nil {
		clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			return nil
		}
		m.c = clientset
	}
	ctx := context.Background()
	node, err := m.c.CoreV1().Nodes().Get(ctx, m.Node, metav1.GetOptions{})
	if err != nil {
		return err
	}
	after := node.DeepCopy()
	taint, err := parseTaint(m.Taint)
	if err != nil {
		return err
	}
	newTaints := []corev1.Taint{}
	for _, t := range after.Spec.Taints {
		if hasTaint(after, taint) {
			continue
		}
		newTaints = append(newTaints, t)
	}
	if len(node.Spec.Taints) == len(newTaints) {
		logrus.Info("removable taint is not found")
		return nil
	}
	after.Spec.Taints = newTaints
	_, err = m.c.CoreV1().Nodes().Update(ctx, after, metav1.UpdateOptions{})
	return err
}

func (m Manager) CanTaintNewNode() error {
	if m.c == nil {
		clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			return nil
		}
		m.c = clientset
	}
	nodes, err := m.c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	taint, err := parseTaint(m.Taint)
	if err != nil {
		return err
	}
	taintedNodeCount := 0
	for _, node := range nodes.Items {
		if hasTaint(&node, taint) {
			taintedNodeCount += 1
		}
	}
	if taintedNodeCount == m.MaxTaintedNodeCount {
		return fmt.Errorf("tainted node count over %d", m.MaxTaintedNodeCount)
	}
	return nil

}

func (m Manager) AddTaint() error {
	if m.DryRun {
		logrus.Infof("dryrun: add taint, %s, %s", m.Node, m.Taint)
		if err := m.NotifySlack(
			fmt.Sprintf("Node *%s* has been tainted", m.Node),
		); err != nil {
			logrus.Error(err)
		}
		return nil
	}
	if reasonWhyNot := m.CanTaintNewNode(); reasonWhyNot != nil {
		logrus.Info("can not taint node")
		logrus.Info(reasonWhyNot)
		return nil

	}
	if m.c == nil {
		clientset, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			return nil
		}
		m.c = clientset
	}
	ctx := context.Background()
	node, err := m.c.CoreV1().Nodes().Get(ctx, m.Node, metav1.GetOptions{})
	if err != nil {
		return err
	}
	after := node.DeepCopy()
	taint, err := parseTaint(m.Taint)
	if err != nil {
		return err
	}
	for _, t := range after.Spec.Taints {
		if hasTaint(after, t) {
			logrus.Info("taint is already added")
			return nil
		}
	}
	after.Spec.Taints = append(after.Spec.Taints, taint)
	_, err = m.c.CoreV1().Nodes().Update(ctx, after, metav1.UpdateOptions{})
	if nerr := m.NotifySlack(
		fmt.Sprintf("Node *%s* has been tainted", m.Node),
	); err != nil {
		logrus.Error(nerr)
	}
	return err
}

func parseTaint(taint string) (corev1.Taint, error) {
	// taint format is key1=value1:NoSchedule
	tmp := strings.Split(taint, "=")
	if len(tmp) != 2 {
		return corev1.Taint{}, fmt.Errorf("format error")
	}
	tmp2 := strings.Split(tmp[1], ":")
	if len(tmp2) != 2 {
		return corev1.Taint{}, fmt.Errorf("format error")
	}
	effect := corev1.TaintEffectNoSchedule
	switch tmp2[1] {
	case "NoSchedule":
		effect = corev1.TaintEffectNoSchedule
	case "NoExecute":
		effect = corev1.TaintEffectNoExecute
	case "PreferNoSchedule":
		effect = corev1.TaintEffectPreferNoSchedule
	default:
		return corev1.Taint{}, fmt.Errorf("invalid effect")

	}
	return corev1.Taint{
		Key:    tmp[0],
		Value:  tmp2[0],
		Effect: effect,
	}, nil
}

func hasTaint(node *corev1.Node, taint corev1.Taint) bool {
	for _, t := range node.Spec.Taints {
		if cmp.Equal(t, taint) {
			return true
		}
	}
	return false
}
