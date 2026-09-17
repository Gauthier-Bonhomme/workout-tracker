package user

import (
	"fmt"

	"github.com/a-h/templ"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// DonneesTableauDeBord rassemble ce que la page d'accueil affiche. Tout est
// prepare par le handler : la vue ne lance aucune requete.
type DonneesTableauDeBord struct {
	User      *database.User
	Resume    *database.Resume
	Annees    []database.ResumeAnnee
	Types     []database.ResumeType
	Dernieres []*database.Workout
	Autres    []*database.User
}

// Periode renvoie l'etendue du carnet, par exemple « 2014 → 2021 ».
func (d DonneesTableauDeBord) Periode() string {
	if len(d.Annees) == 0 {
		return ""
	}

	premiere := d.Annees[0].Annee
	derniere := d.Annees[len(d.Annees)-1].Annee

	if premiere == derniere {
		return premiere
	}

	return premiere + " → " + derniere
}

// hauteurBarre renvoie la hauteur d'une barre de la frise, en pourcentage de la
// plus haute.
//
// La valeur passe par templ.SafeCSS : templ neutralise sinon tout attribut
// style construit a l'execution, et la barre se retrouverait sans hauteur.
func hauteurBarre(valeur, maxi float64) templ.SafeCSS {
	if maxi <= 0 {
		return templ.SafeCSS("height:0%")
	}

	// Un plancher visible : une annee a deux sorties doit rester cliquable.
	part := max(valeur/maxi*100, 4)

	return templ.SafeCSS(fmt.Sprintf("height:%.1f%%", part))
}

// maxDistanceAnnee renvoie la plus grande distance annuelle, qui sert d'echelle.
func maxDistanceAnnee(annees []database.ResumeAnnee) float64 {
	maxi := 0.0
	for _, a := range annees {
		maxi = max(maxi, a.Distance)
	}

	return maxi
}
