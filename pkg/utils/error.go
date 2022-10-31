package utils

import (
	"errors"
	"strings"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type aggregate []error

var _ utilerrors.Aggregate = aggregate{}

// NewMultiLineAggregate returns an aggregate error with multi-line output
func NewMultiLineAggregate(errList []error) error {
	var errs []error
	for _, e := range errList {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return aggregate(errs)
}

// Error is part of the error interface.
func (agg aggregate) Error() string {
	msgs := make([]string, len(agg))
	for i := range agg {
		msgs[i] = agg[i].Error()
	}
	return strings.Join(msgs, "\n")
}

// Errors is part of the Aggregate interface.
func (agg aggregate) Errors() []error {
	return []error(agg)
}

// Is is part of the Aggregate interface
func (agg aggregate) Is(target error) bool {
	return agg.visit(func(err error) bool {
		return errors.Is(err, target)
	})
}

func (agg aggregate) visit(f func(err error) bool) bool {
	for _, err := range agg {
		switch err := err.(type) {
		case aggregate:
			if match := err.visit(f); match {
				return match
			}
		case utilerrors.Aggregate:
			for _, nestedErr := range err.Errors() {
				if match := f(nestedErr); match {
					return match
				}
			}
		default:
			if match := f(err); match {
				return match
			}
		}
	}

	return false
}

// appendErrors append errs, return appended result
func AppendErrors(errsList ...[]error) []error {
	var returnErr []error
	for _, errs := range errsList {
		returnErr = append(returnErr, errs...)
	}
	return returnErr
}
