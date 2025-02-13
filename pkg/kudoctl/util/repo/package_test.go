package repo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/go-test/deep"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func TestReadFileSystemPackage(t *testing.T) {
	// Set Kubernetes random seed for deterministic test results on the name
	rand.Seed(1)

	tests := []struct {
		name         string
		instanceName string
		path         string
		goldenFiles  string
	}{
		{"zookeeper", "zookeeper-xn8fg", "testdata/zk", "testdata/zk-crd-golden1"},
		{"zookeeper", "zookeeper-txhzt", "testdata/zk.tar.gz", "testdata/zk-crd-golden2"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-from-%s", tt.name, tt.path), func(t *testing.T) {
			bundle, err := NewBundle(tt.path)
			if err != nil {
				t.Fatalf("Found unexpected error: %v", err)
			}
			actual, err := bundle.GetCRDs()
			if err != nil {
				t.Fatalf("Found unexpected error: %v", err)
			}
			actual.Instance.ObjectMeta.Name = tt.instanceName
			golden, err := loadCRDsFromPath(tt.goldenFiles)
			if err != nil {
				t.Fatalf("Found unexpected error when loading golden files: %v", err)
			}

			// we need to sort here because current yaml parsing is not preserving the order of fields
			// at the same time, the deep library we use for equality does not support ignoring order
			sort.Slice(actual.OperatorVersion.Spec.Parameters, func(i, j int) bool {
				return actual.OperatorVersion.Spec.Parameters[i].Name < actual.OperatorVersion.Spec.Parameters[j].Name
			})
			sort.Slice(golden.OperatorVersion.Spec.Parameters, func(i, j int) bool {
				return golden.OperatorVersion.Spec.Parameters[i].Name < golden.OperatorVersion.Spec.Parameters[j].Name
			})

			if diff := deep.Equal(golden, actual); diff != nil {
				t.Errorf("%+v\n", diff)
			}
		})
	}
}

func loadCRDsFromPath(goldenPath string) (*PackageCRDs, error) {
	isOperatorFile := func(name string) bool {
		return strings.HasSuffix(name, "operator.golden")
	}

	isVersionFile := func(name string) bool {
		return strings.HasSuffix(name, "operatorversion.golden")
	}

	isInstanceFile := func(name string) bool {
		return strings.HasSuffix(name, "instance.golden")
	}

	result := &PackageCRDs{}
	err := filepath.Walk(goldenPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == goldenPath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		switch {
		case isOperatorFile(info.Name()):
			var f v1alpha1.Operator
			if err = yaml.Unmarshal(bytes, &f); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.Operator = &f
		case isVersionFile(info.Name()):
			var fv v1alpha1.OperatorVersion
			if err = yaml.Unmarshal(bytes, &fv); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.OperatorVersion = &fv
		case isInstanceFile(info.Name()):
			var i v1alpha1.Instance
			if err = yaml.Unmarshal(bytes, &i); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.Instance = &i
		default:
			return fmt.Errorf("unexpected file in the tarball structure %s", info.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
