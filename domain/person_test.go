package domain

// import (
// 	"testing"
// 	"time"
// )

// func TestBuildTreeFromAscendantsAndParentsDescendants(t *testing.T) {
// 	type testArgs struct {
// 		testName          string
// 		person            Person
// 		expectedBuiltTree Person
// 	}

// 	tests := []testArgs{
// 		{
// 			"should use CancellationPolicy's Text when it's filled",
// 			Person{
// 				ID:     "IDCaio",
// 				Name:   "Caio",
// 				Gender: Male,
// 				Ascendants: map[int][]Person{
// 					1: []Person{
// 						{ID: "IDLuis", Name: "Luis", Gender: Male},
// 						{ID: "IDDayse", Name: "Dayse", Gender: Female},
// 					},
// 					2: []Person{
// 						{ID: "IDAlfredo", Name: "Alfredo", Gender: Male},
// 						{ID: "IDZézé", Name: "Zézé", Gender: Female},
// 					},
// 					3: []Person{
// 						{ID: "IDMotherOfAlfredo", Name: "MotherOfAlfredo", Gender: Female},
// 					},
// 				},
// 				Father: &Person{
// 					ID:     "IDLuis",
// 					Name:   "Luis",
// 					Gender: Male,
// 					Descendants: map[int][]Person{
// 						1: []Person{
// 							{ID: "IDCaio", Name: "Caio", Gender: Male},
// 							{ID: "IDVivian", Name: "Vivian", Gender: Female},
// 						},
// 						2: []Person{
// 							{ID: "IDCauê", Name: "Cauê", Gender: Male},
// 						},
// 					},
// 				},
// 			},
// 			time.Time{},
// 			time.Time{},
// 			"Broker cancellation text",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
// 			return func(t *testing.T) {
// 				translator := i18n.NewTranslator(translations.Messages, i18n.PtLocale)
// 				actualPolicyText, err := tt.cancellationPolicy.GetText(*translator, tt.reservationDate, tt.now)

// 				assert.Nil(t, err)
// 				if actualPolicyText != tt.expectedPolicyText {
// 					t.Errorf("actual policy text and expected are different. \n actual: %s \n expected: %s", actualPolicyText, tt.expectedPolicyText)
// 				}
// 			}
// 		}(tt))
// 	}
// }
