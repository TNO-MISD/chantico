package log

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	vol "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/volumes"

	// chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	// utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = k8sruntime.NewScheme()
)

type Info interface {
	String() (string, error)
}

type Markdown interface {
	Markdown()
}

type ClusterState struct {
	ClusterInfo []Info
	CRDStates   []Info
}

func (cs *ClusterState) Markdown() string {
	return "TODO"
}

type ChanticoState struct {
	Error        error
	File         string
	Line         int
	FunctionName string
	Stack        string
}

func (cs *ChanticoState) Markdown() string {
	template := `
- File: %s
- Line: %d
- FunctionName: %s
- stack status:
` + "```" + `
%s
` + "```" + `
`
	return fmt.Sprintf(template, cs.File, cs.Line, cs.FunctionName, cs.Stack)
}

type PostMortem struct {
	Timestamp     time.Time
	ClusterState  ClusterState
	ChanticoState ChanticoState
}

func NewPostMortem(err error, stack string) *PostMortem {
	// Get Chantico current state
	var ok bool
	var pc uintptr
	chanticoState := ChanticoState{Error: err, Stack: stack}

	pc, chanticoState.File, chanticoState.Line, ok = runtime.Caller(1)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	if fn != nil {
		chanticoState.FunctionName = fn.Name()
	}

	// Get the cluster state
	clusterState := ClusterState{}
	chantico.AddToScheme(scheme)

	cfg := ctrl.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil
	}

	measurementDevices := &chantico.MeasurementDeviceList{}
	c.List(context.TODO(), measurementDevices, client.InNamespace("chantico"))
	fmt.Printf("%#v\n", measurementDevices.Items)

	return &PostMortem{
		ChanticoState: chanticoState,
		ClusterState:  clusterState,
	}
}

func (pm *PostMortem) Markdown() string {
	// This template is based on the issue template developped by Jeroen
	template := `
---
name: 🐛 Bug Report
about: Report a problem or unexpected behavior in the system
title: '[BUG] '
labels: bug
assignees: ''

---

## 🐞 Description

A clear and concise description of what the bug is, where it happens, and what the expected behavior should be.

---

## 🔁 How to Reproduce

### Cluster state

%s

### Chantico state

%s

---

## 🧪 Suggested Testing or Validation

Explain how the fix can be verified. Mention test cases, test environments, or steps to revalidate.

---

## 📂 Logs, Screenshots, or Code Snippets

Include logs, screenshots, or relevant code snippets to support the bug report.

---
`
	return fmt.Sprintf(template, pm.ChanticoState.Markdown(), pm.ClusterState.Markdown())
}

func (pm *PostMortem) SaveAndQuit() {
	filename := fmt.Sprintf("%s/bug%d.md", os.Getenv(vol.ChanticoVolumeLocationEnv), pm.Timestamp.UnixMicro())

	err := os.WriteFile(filename, []byte(pm.Markdown()), 0666)
	if err != nil {
		panic(fmt.Sprintf("Could not save postmortem at location %s\nPost mortem content:%s\n", filename, pm.Markdown()))
	}
	panic(fmt.Sprintf("New postmortem generated and saved at %s", filename))
}
