package kudo

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestSimpleK2o() *Client {
	client := Client{}
	client.clientset = fake.NewSimpleClientset()
	return &client
}

func TestNewK2oClient(t *testing.T) {
	tests := []struct {
		err string
	}{
		{"invalid configuration: no configuration has been provided"}, // non existing test
	}

	for _, tt := range tests {
		// Just interested in errors
		_, err := NewClient("default", "")
		if err.Error() != tt.err {
			t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.err, err.Error())
		}
	}
}

func TestK2oClient_CRDsInstalled(t *testing.T) {
	k2o := newTestSimpleK2o()
	err := k2o.CRDsInstalled("default")
	if err != nil {
		t.Errorf("\nexpected: <nil>\n     got: %v", err)
	}
}

func TestK2oClient_OperatorExistsInCluster(t *testing.T) {

	obj := v1alpha1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.Operator
	}{
		{false, "", "", "", nil},               // 1
		{false, "", "default", "default", nil}, // 2
		{true, "", "", "", &obj},               // 3
		{true, "", "default", "", &obj},        // 4
		{false, "", "", "kudo", &obj},          // 4
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		_, err := k2o.clientset.KudoV1alpha1().Operators(tt.createns).Create(tt.obj)
		if err != nil {
			if err.Error() != "object does not implement the Object interfaces" {
				t.Errorf("unexpected error: %+v", err)
			}
		}

		// test if Operator exists in namespace
		exist := k2o.OperatorExistsInCluster("test", tt.getns)

		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_AnyOperatorVersionExistsInCluster(t *testing.T) {
	obj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.OperatorVersion
	}{
		{false, "", "", "", nil},               // 1
		{false, "", "default", "default", nil}, // 2
		{true, "", "", "", &obj},               // 3
		{false, "", "", "qa", &obj},            // 4
		{true, "", "default", "", &obj},        // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create OperatorVersion
		k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Create(tt.obj)

		// test if OperatorVersion exists in namespace
		exist := k2o.AnyOperatorVersionExistsInCluster("test", tt.getns)
		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_InstanceExistsInCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				"operator":                "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	wrongObj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				"operator":                "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-0.9",
			},
		},
	}

	instanceNamespace := "testnamespace"

	tests := []struct {
		name           string
		instanceExists bool
		namespace      string
		instanceName   string
		obj            *v1alpha1.Instance
	}{
		{"no existing instance in cluster", false, "", "", nil},                                                     // 1
		{"same namespace and instance name", true, instanceNamespace, obj.ObjectMeta.Name, &obj},                    // 3
		{"instance with new name", false, instanceNamespace, "nonexisting-instance-name", &obj},                     // 5
		{"same instance name in different namespace", false, "different-namespace", obj.ObjectMeta.Name, &wrongObj}, // 7
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.clientset.KudoV1alpha1().Instances(instanceNamespace).Create(tt.obj)
			if err != nil {
				t.Fatalf("%s: Error during test setup, cannot create test instance %v", tt.name, err)
			}

		}

		// test if OperatorVersion exists in namespace
		exist, _ := k2o.InstanceExistsInCluster("test", tt.namespace, "1.0", tt.instanceName)
		if tt.instanceExists != exist {
			t.Errorf("%s:\nexpected: %v\n     got: %v", tt.name, tt.instanceExists, exist)
		}
	}
}

func TestK2oClient_OperatorVersionInClusterOutOfSync(t *testing.T) {
	obj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-1.0",
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "1.0",
		},
	}

	outdatedObj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-0.9",
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "0.9",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.OperatorVersion
	}{
		{false, "", "", "", nil},                  // 1
		{false, "", "default", "default", nil},    // 2
		{true, "", "", "", &obj},                  // 3
		{true, "", "", "", &obj},                  // 4
		{false, "", "", "qa", &obj},               // 5
		{true, "", "qa", "qa", &obj},              // 6
		{false, "", "kudo", "kudo", &outdatedObj}, // 7
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Instance
		k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Create(tt.obj)

		// test if OperatorVersion exists in namespace
		exist := k2o.OperatorVersionInClusterOutOfSync("test", "1.0", tt.getns)
		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_InstallOperatorObjToCluster(t *testing.T) {
	obj := v1alpha1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.Operator
	}{
		{"", "operators.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "operators.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "operators.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "operators.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                           // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().Operators(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallOperatorObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().Operators(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestK2oClient_InstallOperatorVersionObjToCluster(t *testing.T) {
	obj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.OperatorVersion
	}{
		{"", "operatorversions.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "operatorversions.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "operatorversions.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "operatorversions.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj}, // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallOperatorVersionObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestK2oClient_InstallInstanceObjToCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.Instance
	}{
		{"", "instances.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "instances.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "instances.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "instances.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                           // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().Instances(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallInstanceObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().Instances(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}
