package helpers

import (
	"context"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

func HumanDuration(d time.Duration) string {
	return templatehelpers.HumanDuration(d)
}

func HumanElevation(ctx context.Context, d float64) string {
	return templatehelpers.HumanElevationFor(CurrentUser(ctx).PreferredUnits().Elevation())(d)
}

func HumanDistance(ctx context.Context, d float64) string {
	return templatehelpers.HumanDistanceFor(CurrentUser(ctx).PreferredUnits().Distance(nil))(d)
}

func HumanDistanceForWorkout(ctx context.Context, w *database.Workout, d float64) string {
	if w != nil {
		return templatehelpers.HumanDistanceFor(CurrentUser(ctx).PreferredUnits().Distance(&w.Type))(d)
	}
	return HumanDistance(ctx, d)
}

func DistanceUnitForWorkout(ctx context.Context, w *database.Workout) string {
	if w != nil {
		return CurrentUser(ctx).PreferredUnits().Distance(&w.Type)
	}
	return CurrentUser(ctx).PreferredUnits().Distance(nil)
}

func HumanTempo(ctx context.Context, d float64) string {
	return templatehelpers.HumanTempoFor(CurrentUser(ctx).PreferredUnits().Distance(nil))(d)
}

func HumanTempoForWorkout(ctx context.Context, w *database.Workout, d float64) string {
	if w != nil {
		return templatehelpers.HumanTempoFor(CurrentUser(ctx).PreferredUnits().Tempo(&w.Type))(d)
	}
	return HumanTempo(ctx, d)
}

func TempoUnitForWorkout(ctx context.Context, w *database.Workout) string {
	if w != nil {
		return CurrentUser(ctx).PreferredUnits().Tempo(&w.Type)
	}
	return CurrentUser(ctx).PreferredUnits().Tempo(nil)
}

func HumanSpeed(ctx context.Context, d float64) string {
	return templatehelpers.HumanSpeedFor(CurrentUser(ctx).PreferredUnits().Speed(nil))(d)
}

func HumanSpeedForWorkout(ctx context.Context, w *database.Workout, d float64) string {
	if w != nil {
		return templatehelpers.HumanSpeedFor(CurrentUser(ctx).PreferredUnits().Speed(&w.Type))(d)
	}
	return HumanSpeed(ctx, d)
}

func SpeedUnitForWorkout(ctx context.Context, w *database.Workout) string {
	if w != nil {
		return CurrentUser(ctx).PreferredUnits().Speed(&w.Type)
	}
	return CurrentUser(ctx).PreferredUnits().Speed(nil)
}

func HumanCadence(d float64) string {
	return templatehelpers.HumanCadence(d)
}

func HumanPower(d float64) string {
	return templatehelpers.HumanPower(d)
}

func HumanCalories(d float64) string {
	return templatehelpers.HumanCalories(d)
}

func HumanWeight(ctx context.Context, d float64) string {
	return templatehelpers.HumanWeightFor(CurrentUser(ctx).PreferredUnits().Weight())(d)
}

func HumanHeight(ctx context.Context, d float64) string {
	return templatehelpers.HumanHeightFor(CurrentUser(ctx).PreferredUnits().Height())(d)
}

func HumanHeightSingle(ctx context.Context, d float64) string {
	return templatehelpers.HumanHeightSingleFor(CurrentUser(ctx).PreferredUnits().Height())(d)
}

// espaceFine separe les groupes de milliers. U+202F ne se casse pas en fin de
// ligne, contrairement a une espace ordinaire.
const espaceFine = "\u202f"

// Total remet en forme un chiffre de total.
//
// Passe la centaine, les centiemes ne disent plus rien ; passe le millier,
// l'oeil a besoin des groupes de trois. « 93 347 » se lit d'un coup d'oeil la ou
// « 93347.28 » demande de compter les chiffres. En dessous de cent, la valeur
// est laissee telle quelle -- sur une sortie de 47,08 km, la decimale compte.
func Total(valeur string) string {
	signe, entier, _ := decomposeNombre(valeur)
	if entier == "" || len(entier) < 3 {
		return valeur
	}

	if len(entier) < 4 {
		return signe + entier
	}

	groupes := []string{}
	for len(entier) > 3 {
		groupes = append([]string{entier[len(entier)-3:]}, groupes...)
		entier = entier[:len(entier)-3]
	}

	return signe + strings.Join(append([]string{entier}, groupes...), espaceFine)
}

// decomposeNombre separe le signe et la partie entiere d'un nombre deja mis en
// forme. L'entier est vide des que la chaine n'en est pas un : « N/A », un
// tempo « 4:37 », une duree « 3d 17h » ressortent alors intacts.
func decomposeNombre(valeur string) (string, string, string) {
	entier, fraction, _ := strings.Cut(valeur, ".")

	signe := ""
	if reste, ok := strings.CutPrefix(entier, "-"); ok {
		signe, entier = "-", reste
	}

	if entier == "" {
		return signe, "", fraction
	}

	for _, r := range entier {
		if r < '0' || r > '9' {
			return signe, "", fraction
		}
	}

	return signe, entier, fraction
}
