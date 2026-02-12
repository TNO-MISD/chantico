package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"example.com/m/internal/graph"
)

type KubernetesClient struct {
	clientset     *kubernetes.Clientset
	dynamicClient *dynamic.DynamicClient
}

func New(kubeconfigPath string) (*KubernetesClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamic, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesClient{
		clientset:     cs,
		dynamicClient: dynamic,
	}, nil
}

type Pod struct {
	Namespace string
	Name      string
}

func (k *KubernetesClient) GetPods() ([]Pod, error) {
	pods, err := k.clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	p := make([]Pod, 0, pods.Size())

	for _, pod := range pods.Items {
		p = append(p, Pod{Namespace: pod.Namespace, Name: pod.Name})
	}
	return p, nil
}

func (k *KubernetesClient) GetNamespaces() (*corev1.NamespaceList, error) {
	return k.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
}

func (k *KubernetesClient) GetDataCenterResources() ([]*graph.Node, error) {
	gvr := schema.GroupVersionResource{
		Group:    "chantico.ci.tno.nl",
		Version:  "v1alpha1",
		Resource: "datacenterresources",
	}

	objects, err := k.dynamicClient.Resource(gvr).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var output []*graph.Node

	for _, object := range objects.Items {
		name := object.GetNamespace() + "-" + object.GetName()
		node := graph.Node{
			Name: name,
		}

		parents, found, err := unstructured.NestedStringSlice(
			object.Object,
			"spec",
			"parent",
		)
		if err != nil {
			return nil, err
		}
		if found {
			for _, parent := range parents {
				p := &graph.Node{
					Name: object.GetNamespace() + "-" + parent,
				}
				node.Parents = append(node.Parents, p)
			}
		}
		output = append(output, &node)
	}

	return output, nil
}
