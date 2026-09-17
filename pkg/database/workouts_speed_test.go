package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// traceReguliere fabrique une trace echantillonnee a 1 Hz parcourue a vitesse
// constante.
func traceReguliere(nombre int, vitesse float64) []MapPoint {
	points := make([]MapPoint, nombre)
	for i := 1; i < nombre; i++ {
		points[i] = MapPoint{Distance: vitesse, Duration: time.Second}
	}

	return points
}

func TestVitessesLissees_AllureConstante(t *testing.T) {
	t.Parallel()

	v := VitessesLissees(traceReguliere(120, 3))

	for i, vitesse := range v {
		assert.InDelta(t, 3.0, vitesse, 0.01, "point %d", i)
	}
}

func TestVitesseMaximale_EcarteUnSautDePosition(t *testing.T) {
	t.Parallel()

	points := traceReguliere(120, 3)
	// Un point atterrit a 200 m de sa place : brut, l'intervalle vaut 720 km/h.
	points[60].Distance = 200

	assert.InDelta(t, 200.0, points[60].AverageSpeed(), 0.01)
	assert.InDelta(t, 3.0, VitesseMaximale(points), 0.05)
}

func TestVitesseMaximale_ConserveUneAccelerationReelle(t *testing.T) {
	t.Parallel()

	// Trente secondes a 3 m/s, puis trente a 9 m/s : l'acceleration est tenue,
	// ce n'est pas une aberration isolee.
	points := append(traceReguliere(60, 3), traceReguliere(60, 9)[1:]...)

	assert.InDelta(t, 9.0, VitesseMaximale(points), 0.05)
}

func TestVitesseMaximale_SuitUnSprint(t *testing.T) {
	t.Parallel()

	// Un demarrage de 3 a 8 m/s en quatre secondes, tenu dix secondes : le
	// rythme sort de l'ordinaire de la sortie, mais l'acceleration reste dans
	// les cordes d'un coureur. La pointe doit ressortir.
	points := traceReguliere(120, 3)
	for i, v := range []float64{4.25, 5.5, 6.75, 8} {
		points[60+i].Distance = v
	}

	for i := 64; i < 70; i++ {
		points[i].Distance = 8
	}

	assert.Greater(t, VitesseMaximale(points), 7.0)
}

func TestVitessesLissees_TracesTropCourtes(t *testing.T) {
	t.Parallel()

	assert.Empty(t, VitessesLissees(nil))
	assert.Len(t, VitessesLissees(traceReguliere(1, 3)), 1)
	assert.Zero(t, VitesseMaximale(nil))
}

func TestVitesseMaximale_IgnoreLesDureesNulles(t *testing.T) {
	t.Parallel()

	points := traceReguliere(60, 2)
	// Deux releves partagent le meme horodatage : la division serait infinie.
	points[30] = MapPoint{Distance: 5, Duration: 0}

	v := VitesseMaximale(points)
	assert.Positive(t, v)
	assert.Less(t, v, 10.0)
}
