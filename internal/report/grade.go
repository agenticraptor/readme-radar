package report

// Grade is a letter grade A–F derived from a 0–100 score.
type Grade string

const (
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
	GradeF Grade = "F"
)

// GradeFor maps a 0–100 score onto a letter grade.
func GradeFor(score int) Grade {
	switch {
	case score >= 85:
		return GradeA
	case score >= 70:
		return GradeB
	case score >= 55:
		return GradeC
	case score >= 40:
		return GradeD
	default:
		return GradeF
	}
}

// VerdictFor maps a 0–100 score onto the headline recommendation.
func VerdictFor(score int) Verdict {
	switch {
	case score >= 70:
		return VerdictTrust
	case score >= 45:
		return VerdictCaution
	default:
		return VerdictAvoid
	}
}
