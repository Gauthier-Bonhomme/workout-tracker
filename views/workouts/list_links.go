package workouts

import (
	"net/url"
	"strconv"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// LienListe construit l'adresse de la liste en conservant les criteres en cours
// et en remplacant celui qu'on change.
//
// Sans cela, choisir une annee effacerait le sport selectionne, et tourner la
// page effacerait les deux.
func LienListe(base string, f *database.WorkoutFilters, clef, valeur string) string {
	q := url.Values{}

	poser := func(nom, actuel string) {
		if clef == nom {
			if valeur != "" {
				q.Set(nom, valeur)
			}

			return
		}

		if actuel != "" {
			q.Set(nom, actuel)
		}
	}

	poser("type", f.Type.String())
	poser("year", f.Year)
	poser("order_by", f.OrderBy)
	poser("order_dir", f.OrderDir)

	if f.Since != "" && f.Since != "forever" {
		poser("since", f.Since)
	}

	// Changer un critere ramene a la premiere page : rester en page 12 d'une
	// selection qui n'en compte que trois afficherait une liste vide.
	if clef == "page" && valeur != "" && valeur != "1" {
		q.Set("page", valeur)
	}

	if len(q) == 0 {
		return base
	}

	return base + "?" + q.Encode()
}

// LienPage renvoie l'adresse d'une page de resultats.
func LienPage(base string, f *database.WorkoutFilters, page int) string {
	return LienListe(base, f, "page", strconv.Itoa(page))
}

// Pages renvoie le nombre de pages du resultat courant.
func Pages(total int64) int {
	return database.NombreDePages(total)
}
