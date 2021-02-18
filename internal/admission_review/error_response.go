package admission_review

import (
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ErrorResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Status:  "Failure",
			Code:    500,
			Message: err.Error(),
		},
	}
}
