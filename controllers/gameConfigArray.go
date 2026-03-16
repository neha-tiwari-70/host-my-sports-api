package controllers

import "fmt"

type SuperGameCategory struct {
	Name            string
	HasSetNo        bool
	HasPlayerId     bool
	HasScoredAt     bool
	HasIsPenalty    bool
	HasPointsScored bool
	HasDistance     bool
	CalculateScore  bool
}

// GetSuperGameCategories returns a **copy** of the array
func GetSuperGameCategory(name string, subCategory string, tournamentType string) (SuperGameCategory, error) {

	var EmptySuperCategory SuperGameCategory
	var superGameCategoryArray []SuperGameCategory = []SuperGameCategory{
		{
			Name:            "GoalOriented",
			HasSetNo:        false,
			HasPlayerId:     true,
			HasScoredAt:     true,
			HasIsPenalty:    true,
			HasPointsScored: true,
			CalculateScore:  true,
			HasDistance:     false,
		}, {
			Name:            "SetOriented",
			HasSetNo:        true,
			HasPlayerId:     false,
			HasScoredAt:     false,
			HasIsPenalty:    false,
			HasPointsScored: true,
			CalculateScore:  true,
			HasDistance:     false,
		}, {
			Name:            "PointBased",
			HasSetNo:        false,
			HasPlayerId:     false,
			HasScoredAt:     false,
			HasIsPenalty:    false,
			HasPointsScored: false,
			CalculateScore:  true,
			HasDistance:     false,
		}, {
			Name:            "DirectOutcome",
			HasSetNo:        false,
			HasPlayerId:     false,
			HasScoredAt:     false,
			HasIsPenalty:    false,
			HasPointsScored: false,
			CalculateScore:  false,
			HasDistance:     false,
		}, {
			Name:            "Athletics",
			HasSetNo:        false,
			HasPlayerId:     false,
			HasScoredAt:     false,
			HasIsPenalty:    false,
			HasPointsScored: false,
			CalculateScore:  true,
			HasDistance:     false,
		},
	}

	// for i := range superGameCategoryArray {
	// 	if superGameCategoryArray[i].Name == name {
	// 		return superGameCategoryArray[i]
	// 	}
	// }

	for _, category := range superGameCategoryArray {
		if category.Name == name {
			// Special handling for Athletics subcategory
			if name == "Athletics" {
				switch subCategory {
				case "TimeBased":
					category.HasScoredAt = true
					category.HasPointsScored = false
					category.HasDistance = false
				case "DistanceBased":
					category.HasScoredAt = false
					category.HasPointsScored = false
					category.HasDistance = true
				case "HeightBased":
					category.HasScoredAt = false
					category.HasPointsScored = false
					category.HasDistance = true
				case "RankBased":
					category.HasScoredAt = false
					category.HasPointsScored = true
					category.HasDistance = false
				}
			}
			return category, nil
		}
	}

	for i := range superGameCategoryArray {
		if superGameCategoryArray[i].Name == name {
			return superGameCategoryArray[i], nil
		}
	}
	return EmptySuperCategory, fmt.Errorf("No superGameCategory found with the name '" + name + "'")
}
