package service

import (
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

func serviceExists(name, namespace string, lister listerscorev1.ServiceLister) (bool, error) {
	_, err := lister.Services(namespace).Get(name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get service %q from lister: %v", name, err)
	}
	return true, nil
}
