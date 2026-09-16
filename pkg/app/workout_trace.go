package app

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v5"
	"github.com/spf13/cast"
)

// Dimensions de la vignette de trace, en unites du viewBox : le SVG est mis a
// l'echelle par le CSS, ces valeurs ne fixent que le rapport d'aspect.
const (
	traceLargeur = 232.0
	traceHauteur = 152.0
	traceMarge   = 10.0
	// Au-dela, la vignette ne gagne plus en precision mais alourdit la reponse.
	tracePointsMax = 220
)

// workoutsTraceHandler rend le parcours d'une seance en SVG.
//
// La liste des seances ne precharge que map_data, jamais les points GPS : les
// charger pour toutes les seances ferait exploser la memoire. La vignette est
// donc servie a part, et les cartes de la liste la demandent en lazy loading,
// ce qui limite le cout aux seances reellement affichees a l'ecran.
func (a *App) workoutsTraceHandler(c *echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	w, err := a.getCurrentUser(c).GetWorkout(a.db.Preload("Data.Details"), id)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	if w.Data == nil || w.Data.Details == nil || len(w.Data.Details.Points) < 2 {
		return c.NoContent(http.StatusNoContent)
	}

	// Le contenu d'une seance ne change pas une fois calculee : on laisse le
	// navigateur garder la vignette, sinon chaque defilement relit les points.
	c.Response().Header().Set("Cache-Control", "private, max-age=604800")

	return c.Blob(http.StatusOK, "image/svg+xml", []byte(traceSVG(w.Data.Details.Points)))
}

// traceSVG projette les points sur le plan et renvoie une polyligne.
//
// La couleur est figee plutot que prise dans les variables de theme : un SVG
// charge via <img> n'herite pas du CSS de la page. L'accent choisi tient sur
// fond clair comme sur fond sombre, et le fond du dessin reste transparent.
func traceSVG(points []database.MapPoint) string {
	pas := 1
	if len(points) > tracePointsMax {
		pas = len(points) / tracePointsMax
	}

	retenus := make([]database.MapPoint, 0, tracePointsMax+1)

	for i := 0; i < len(points); i += pas {
		if points[i].Lat != 0 || points[i].Lng != 0 {
			retenus = append(retenus, points[i])
		}
	}

	if len(retenus) < 2 {
		return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 232 152"></svg>`
	}

	// Projection equirectangulaire : a l'echelle d'une sortie, la correction en
	// cosinus de latitude suffit a garder des proportions justes.
	var sommeLat float64
	for _, p := range retenus {
		sommeLat += p.Lat
	}

	k := math.Cos(sommeLat / float64(len(retenus)) * math.Pi / 180)

	xs := make([]float64, len(retenus))
	ys := make([]float64, len(retenus))

	for i, p := range retenus {
		xs[i] = p.Lng * k
		ys[i] = -p.Lat
	}

	minX, maxX := minMax(xs)
	minY, maxY := minMax(ys)

	dx := math.Max(maxX-minX, 1e-9)
	dy := math.Max(maxY-minY, 1e-9)

	echelle := math.Min((traceLargeur-2*traceMarge)/dx, (traceHauteur-2*traceMarge)/dy)
	decX := (traceLargeur-dx*echelle)/2 - minX*echelle
	decY := (traceHauteur-dy*echelle)/2 - minY*echelle

	var b strings.Builder

	b.Grow(len(retenus) * 14)

	for i := range retenus {
		if i > 0 {
			b.WriteByte(' ')
		}

		fmt.Fprintf(&b, "%.1f,%.1f", xs[i]*echelle+decX, ys[i]*echelle+decY)
	}

	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" role="img">`+
			`<polyline points="%s" fill="none" stroke="#E2571E" stroke-width="2.4" `+
			`stroke-linecap="round" stroke-linejoin="round"/>`+
			`<circle cx="%.1f" cy="%.1f" r="3.2" fill="none" stroke="#E2571E" stroke-width="2"/>`+
			`</svg>`,
		traceLargeur, traceHauteur, b.String(),
		xs[0]*echelle+decX, ys[0]*echelle+decY)
}

func minMax(v []float64) (float64, float64) {
	bas, haut := v[0], v[0]

	for _, x := range v[1:] {
		if x < bas {
			bas = x
		}

		if x > haut {
			haut = x
		}
	}

	return bas, haut
}
