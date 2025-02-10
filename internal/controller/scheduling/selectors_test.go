package scheduling

import (
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func TestHasSelector(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		want     bool
	}{
		{
			name: "Deployment with Selector",
			resource: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "example",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Deployment without Selector",
			resource: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Selector: nil,
				},
			},
			want: false,
		},
		{
			name: "StatefulSet with Selector",
			resource: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "example",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "StatefulSet without Selector",
			resource: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Selector: nil,
				},
			},
			want: false,
		},
		{
			name:     "Invalid resource type",
			resource: &struct{}{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSelector(tt.resource); got != tt.want {
				t.Errorf("HasSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckNodeSelector(t *testing.T) {
	tests := []struct {
		name      string
		selector  *metav1.LabelSelector
		procedure k8smanagersv1.Procedure
		wlName    string
		wantErr   bool
	}{
		{
			name: "Selector with matching key and value",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"node-type": "initial",
				},
			},
			procedure: k8smanagersv1.Procedure{
				Selector: k8smanagersv1.Selector{
					Key:     "node-type",
					Initial: "initial",
				},
			},
			wlName:  "workload1",
			wantErr: false,
		},
		{
			name: "Selector without matching key and value",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"node-type": "initial",
				},
			},
			procedure: k8smanagersv1.Procedure{
				Selector: k8smanagersv1.Selector{
					Key:     "node-type",
					Initial: "non-matching-value",
				},
			},
			wlName:  "workload1",
			wantErr: false,
		},
		{
			name:      "Nil selector",
			selector:  nil,
			procedure: k8smanagersv1.Procedure{},
			wlName:    "workload1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckNodeSelector(tt.selector, tt.procedure, tt.wlName); (err != nil) != tt.wantErr {
				t.Errorf("CheckNodeSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateNodeSelector(t *testing.T) {
	key := "disktype"
	value := "ssd"
	expectedNodeSelector := map[string]string{
		key: value,
	}

	nodeSelector := CreateNodeSelector(key, value)

	if !reflect.DeepEqual(nodeSelector, expectedNodeSelector) {
		t.Errorf("CreateNodeSelector() = %v, want %v", nodeSelector, expectedNodeSelector)
	}
}
