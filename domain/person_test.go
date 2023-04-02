package domain

func buildFamilyGraph() FamilyGraph {
	livia := &Person{ID: "IDLivia", Name: "Livia", Gender: Female}
	claudia := &Person{ID: "IDClaudia", Name: "Claudia", Gender: Female}
	zeze := &Person{ID: "IDZézé", Name: "Zézé", Gender: Female}
	luis := &Person{ID: "IDLuis", Name: "Luis", Gender: Male}
	dayse := &Person{ID: "IDDayse", Name: "Dayse", Gender: Female}
	vivian := &Person{ID: "IDVivian", Name: "Vivian", Gender: Female}
	caio := &Person{ID: "IDCaio", Name: "Caio", Gender: Male}
	caua := &Person{ID: "IDCauã", Name: "Cauã", Gender: Male}

	livia.Parents = []*Person{claudia}

	claudia.Children = []*Person{livia}
	claudia.Parents = []*Person{zeze}

	zeze.Children = []*Person{luis, claudia}

	luis.Children = []*Person{caio, vivian}
	luis.Parents = []*Person{zeze}

	dayse.Children = []*Person{caio, vivian}

	caio.Parents = []*Person{dayse, luis}

	vivian.Parents = []*Person{dayse, luis}
	vivian.Children = []*Person{caua}

	caua.Parents = []*Person{vivian}

	return FamilyGraph{
		Members: map[string]*Person{
			caio.ID:    caio,
			livia.ID:   livia,
			claudia.ID: claudia,
			zeze.ID:    zeze,
			luis.ID:    luis,
			dayse.ID:   dayse,
			vivian.ID:  vivian,
			caua.ID:    caua,
		}}
}

// func TestBuildFamilyRelationships(t *testing.T) {
// 	type testArgs struct {
// 		testName          string
// 		person            Person
// 		expectedBuiltTree Person
// 	}

// 	tests := []testArgs{
// 		{
// 			"should get relationships sucessfully",
// 			FamilyGraph{
// 				Members: map[string]*Person{
// 					"IDCaio": &caio,
// 				},
// 				ID:     "IDCaio",
// 				Name:   "Caio",
// 				Gender: Male,
// 				// Ascendants: map[int][]Person{
// 				// 	1: []Person{
// 				// 		{ID: "IDLuis", Name: "Luis", Gender: Male},
// 				// 		{ID: "IDDayse", Name: "Dayse", Gender: Female},
// 				// 	},
// 				// 	2: []Person{
// 				// 		{ID: "IDAlfredo", Name: "Alfredo", Gender: Male},
// 				// 		{ID: "IDZézé", Name: "Zézé", Gender: Female},
// 				// 	},
// 				// 	3: []Person{
// 				// 		{ID: "IDMotherOfAlfredo", Name: "MotherOfAlfredo", Gender: Female},
// 				// 	},
// 				// },
// 				// Father: &Person{
// 				// 	ID:     "IDLuis",
// 				// 	Name:   "Luis",
// 				// 	Gender: Male,
// 				// 	Descendants: map[int][]Person{
// 				// 		1: []Person{
// 				// 			{ID: "IDCaio", Name: "Caio", Gender: Male},
// 				// 			{ID: "IDVivian", Name: "Vivian", Gender: Female},
// 				// 		},
// 				// 		2: []Person{
// 				// 			{ID: "IDCauê", Name: "Cauê", Gender: Male},
// 				// 		},
// 				// 	},
// 				// },
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
