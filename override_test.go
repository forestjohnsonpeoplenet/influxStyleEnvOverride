package influxStyleEnvOverride

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type ExampleObject struct {
	A      string
	Other  *ExampleSubObject
	Others []ExampleSubObject
}

type ExampleSubObject struct {
	Integer    int
	B          string
	unexported int
	Thing      interface{}
	Things     []interface{}
}

type mockKeyValueRetriever struct {
	KeyValues map[string]string
}

func (this mockKeyValueRetriever) get(key string) string {
	return this.KeyValues[key]
}

type testCase struct {
	mutateExampleObject func(*ExampleObject)
	environment         map[string]string
	expectedError       string
}

func TestApplyEnvOverridesBasic(t *testing.T) {
	toTest := testCase{
		mutateExampleObject: func(example *ExampleObject) {
			example.A = "asd2"
			example.Other.B = "asd2"
			example.Other.Integer = 10
			things0 := example.Others[0].Things[0].(ExampleSubObject)
			things0.B = "asd2"
			example.Others[0].B = "asd2"
			thing := example.Others[0].Thing.(ExampleSubObject)
			thing.B = "asd2"
		},
		environment: map[string]string{
			"TEST_A":                   "asd2",
			"TEST_OTHER_B":             "asd2",
			"TEST_OTHER_INTEGER":       "10",
			"TEST_OTHERS_0_THINGS_0_B": "asd2",
			"TEST_OTHERS_0_B":          "asd2",
			"TEST_OTHERS_0_THING_B":    "asd2",
		},
	}

	toTest.execute(t)
}

func TestApplyEnvOverridesWithInvalidInteger(t *testing.T) {
	toTest := testCase{
		mutateExampleObject: func(example *ExampleObject) {
		},
		environment: map[string]string{
			"TEST_OTHER_INTEGER": "o no",
		},
		expectedError: "failed to apply TEST_OTHER_INTEGER to Integer",
	}

	toTest.execute(t)
}

func TestApplyEnvOverridesWithUnsettableField(t *testing.T) {
	toTest := testCase{
		mutateExampleObject: func(example *ExampleObject) {
		},
		environment: map[string]string{
			"TEST_OTHERS_0_UNEXPORTED": "o no",
		},
		expectedError: "is not settable",
	}

	toTest.execute(t)
}

// Note currently this test is expected to fail. Haven't implemented additional slice elements yet.
func TestApplyEnvOverridesWithNonExistentObject(t *testing.T) {
	toTest := testCase{
		mutateExampleObject: func(example *ExampleObject) {
			example.Others = append(example.Others, ExampleSubObject{
				B: "asd2",
			})
		},
		environment: map[string]string{
			"TEST_OTHERS_1_B": "asd2",
		},
	}

	toTest.execute(t)
}

func (this testCase) execute(t *testing.T) {
	ExampleObjectUnderTest := newExampleObject()
	ExampleObjectForComparison := newExampleObject()

	exampleKeyValueRetriever := mockKeyValueRetriever{
		KeyValues: this.environment,
	}
	err := applyEnvOverrides(exampleKeyValueRetriever, "TEST", reflect.ValueOf(&ExampleObjectUnderTest))

	this.mutateExampleObject(&ExampleObjectForComparison)

	if err != nil || this.expectedError != "" {
		actualError := ""
		if err != nil {
			actualError = err.Error()
		}
		if !strings.Contains(actualError, this.expectedError) || this.expectedError == "" {
			expectedErrorDisplay := "nil"
			if this.expectedError != "" {
				expectedErrorDisplay = this.expectedError
			}
			t.Errorf("Expected Error: %s, Actual Error: %s", expectedErrorDisplay, actualError)
			return
		}
	}

	jsonA, err := json.MarshalIndent(ExampleObjectUnderTest, "", "  ")
	if err != nil {
		t.Error(err)
	}
	jsonB, err := json.MarshalIndent(ExampleObjectForComparison, "", "  ")
	if err != nil {
		t.Error(err)
	}

	if string(jsonA) != string(jsonB) {
		t.Errorf("Expected Value: \n%s,\n Actual Value: \n%s\n\n", string(jsonB), string(jsonA))
	}
}

func newExampleObject() ExampleObject {
	toReturn := ExampleObject{
		A: "asd",
		Other: &ExampleSubObject{
			B:       "bsd",
			Integer: 2,
		},
		Others: []ExampleSubObject{
			ExampleSubObject{
				B: "bsd",
				Thing: ExampleSubObject{
					B: "bsd",
				},
			},
		},
	}

	toReturn.Others[0].Things = make([]interface{}, 0)

	toReturn.Others[0].Things = append(
		toReturn.Others[0].Things,
		ExampleSubObject{
			B: "bsd",
		},
	)

	return toReturn
}
