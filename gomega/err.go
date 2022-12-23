package gomega

import (
	"errors"
	"fmt"
	"github.com/onsi/gomega/types"
)

type errorMatcher func(actual error) (success bool, failureMessage string, negativeFailureMessage string)

func (e errorMatcher) Match(actual interface{}) (success bool, err error) {
	if err, ok := actual.(error); ok {
		result, _, _ := e(err)
		return result, nil
	}

	return false, errors.New("expected error type")
}

func (e errorMatcher) FailureMessage(actual interface{}) (message string) {
	if err, ok := actual.(error); ok {
		_, m, _ := e(err)
		return m
	}

	return "Unsupported type"
}

func (e errorMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if err, ok := actual.(error); ok {
		_, _, m := e(err)
		return m
	}

	return "Unsupported type"
}

// WrapError matches if given error is wrapped by actual error
func WrapError(err error) types.GomegaMatcher {
	return errorMatcher(func(actual error) (bool, string, string) {
		return errors.Is(actual, err),
			fmt.Sprintf(`Expected err("%v") to wrap err("%v") but it doesn't'`, actual, err),
			fmt.Sprintf(`Expected err("%v") not to wrap err("%v") but it does`, actual, err)
	})
}
