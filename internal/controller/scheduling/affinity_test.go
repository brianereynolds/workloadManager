package scheduling

import (
	k8smanagersv1 "greyridge.com/workloadManager/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestHasAffinity(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		want     bool
	}{
		{
			name: "Deployment with Node Affinity",
			resource: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "key1",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"value1"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Deployment without Node Affinity",
			resource: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: nil,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "StatefulSet with Node Affinity",
			resource: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "key1",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"value1"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "StatefulSet without Node Affinity",
			resource: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: nil,
						},
					},
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
			if got := HasAffinity(tt.resource); got != tt.want {
				t.Errorf("HasAffinity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckNodeAffinity(t *testing.T) {
	tests := []struct {
		name      string
		affinity  *corev1.NodeAffinity
		procedure k8smanagersv1.Procedure
		wlName    string
		wantErr   bool
	}{
		{
			name: "Affinity with matching key and value",
			affinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "node-type",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"initial"},
								},
							},
						},
					},
				},
			},
			procedure: k8smanagersv1.Procedure{
				Affinity: k8smanagersv1.Affinity{
					Key:     "node-type",
					Initial: "initial",
				},
			},
			wlName:  "workload1",
			wantErr: false,
		},
		{
			name: "Affinity without matching key and value",
			affinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "node-type",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"initial"},
								},
							},
						},
					},
				},
			},
			procedure: k8smanagersv1.Procedure{
				Affinity: k8smanagersv1.Affinity{
					Key:     "node-type",
					Initial: "non-matching-value",
				},
			},
			wlName:  "workload1",
			wantErr: false,
		},
		{
			name:      "Nil affinity",
			affinity:  nil,
			procedure: k8smanagersv1.Procedure{},
			wlName:    "workload1",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckNodeAffinity(tt.affinity, tt.procedure, tt.wlName); (err != nil) != tt.wantErr {
				t.Errorf("CheckNodeAffinity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateNodeAffinity(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  *corev1.NodeAffinity
	}{
		{
			name:  "Valid key and value",
			key:   "key1",
			value: "value1",
			want: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "key1",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"value1"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateNodeAffinity(tt.key, tt.value)
			if got == nil || got.RequiredDuringSchedulingIgnoredDuringExecution == nil {
				t.Errorf("CreateNodeAffinity() = %v, want %v", got, tt.want)
				return
			}

			gotTerms := got.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			wantTerms := tt.want.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

			if len(gotTerms) != len(wantTerms) {
				t.Errorf("CreateNodeAffinity() = %v, want %v", got, tt.want)
				return
			}

			for i, gotTerm := range gotTerms {
				wantTerm := wantTerms[i]
				if len(gotTerm.MatchExpressions) != len(wantTerm.MatchExpressions) {
					t.Errorf("CreateNodeAffinity() = %v, want %v", got, tt.want)
					return
				}

				for j, gotExpr := range gotTerm.MatchExpressions {
					wantExpr := wantTerm.MatchExpressions[j]
					if gotExpr.Key != wantExpr.Key || gotExpr.Operator != wantExpr.Operator || len(gotExpr.Values) != len(wantExpr.Values) {
						t.Errorf("CreateNodeAffinity() = %v, want %v", got, tt.want)
						return
					}

					for k, gotValue := range gotExpr.Values {
						wantValue := wantExpr.Values[k]
						if gotValue != wantValue {
							t.Errorf("CreateNodeAffinity() = %v, want %v", got, tt.want)
						}
					}
				}
			}
		})
	}
}
