package database

import (
	"math"
	"sort"
	"time"
)

// Filtrage des vitesses GPS.
//
// Une trace enregistree a 1 Hz par un telephone comporte des sauts de position :
// un point atterrit a cent metres de sa place, puis la trace reprend son cours.
// Calculee d'un point au suivant, la vitesse y depasse couramment 300 km/h.
// Mesure sur l'export Strava de cette installation : 44 % des sorties portent au
// moins un intervalle au-dessus de 300 km/h, la pire a 9 831 km/h.
//
// Le traitement se fait en deux temps :
//
//  1. filtre de Hampel sur la serie des vitesses instantanees. Un intervalle
//     dont la vitesse s'ecarte de la mediane locale de plus de trois ecarts
//     robustes (estimes par la MAD) et qui suppose en plus une acceleration
//     hors de portee d'un humain est tenu pour une aberration : sa distance est
//     ramenee a ce que le rythme local laissait attendre ;
//  2. moyenne sur une fenetre glissante d'au moins cinq secondes, comme le fait
//     un compteur de velo.
//
// Les deux etapes sont necessaires : le filtre seul laisse encore 15 % des
// sorties trop haut (un saut etale sur plusieurs points ne ressort pas comme une
// aberration isolee), la fenetre seule ne fait que diluer un saut de 200 m sur
// cinq secondes, ce qui donne encore 140 km/h.
//
// Calibrage sur 82 sorties de l'export, en prenant pour reference le
// <MaximumSpeed> que Strava inscrit lui-meme dans ses fichiers TCX :
//
//	calcul brut          mediane  +494 %   pire valeur  9 831 km/h
//	fenetre 5 s seule    mediane   +63 %   pire valeur  1 985 km/h
//	Hampel seul          mediane   +13 %   pire valeur    154 km/h
//	Hampel + fenetre     mediane  +0,2 %   pire valeur    141 km/h (valeur juste)
//
// Apres traitement, plus aucune sortie ne depasse 1,5 fois la valeur de
// reference.
const (
	// Demi-largeur de la fenetre du filtre, en points. A 1 Hz cela fait une
	// trentaine de secondes : assez pour que la mediane decrive l'allure du
	// moment, assez court pour suivre une acceleration reelle.
	hampelDemiFenetre = 15

	// Nombre d'ecarts robustes au-dela duquel un intervalle est ecarte.
	hampelSeuil = 3.0

	// Plancher de l'ecart robuste, en m/s. Sur une allure tres reguliere la MAD
	// tombe a zero et le moindre a-coup passerait pour une aberration.
	hampelPlancher = 0.5

	// Duree minimale de la fenetre de lissage. En dessous, un saut residuel
	// ressort encore ; au-dessus, une pointe reelle de sprint est rabotee.
	fenetreLissage = 5 * time.Second

	// Acceleration au-dela de laquelle un intervalle est tenu pour impossible,
	// en m/s2. Un cycliste ou un coureur ne gagnent pas 3 m/s en une seconde ;
	// un point mal place, si. Ce garde-fou empeche le filtre de raboter une
	// acceleration reelle, qui monte progressivement d'un releve au suivant.
	accelerationMax = 3.0
)

// pasVitesses renvoie la vitesse instantanee de chaque intervalle, en m/s.
// L'element i porte sur l'intervalle entre le point i-1 et le point i ; le
// premier vaut 0.
func pasVitesses(points []MapPoint) []float64 {
	v := make([]float64, len(points))

	for i := range points {
		if s := points[i].Duration.Seconds(); s > 0 {
			v[i] = points[i].Distance / s
		}
	}

	return v
}

// distancesFiltrees renvoie la distance retenue pour chaque intervalle, les
// sauts de position ramenes au rythme local.
//
// Un intervalle n'est ecarte que s'il est a la fois aberrant au regard de
// l'allure du moment et physiquement impossible : sans cette seconde condition,
// un demarrage tenu quelques secondes serait pris pour une erreur, la mediane
// de la fenetre restant celle de l'allure de croisiere.
func distancesFiltrees(points []MapPoint) []float64 {
	vitesses := pasVitesses(points)
	distances := make([]float64, len(points))

	for i := range points {
		distances[i] = points[i].Distance
	}

	if len(points) < 2*hampelDemiFenetre {
		return distances
	}

	fenetre := make([]float64, 0, 2*hampelDemiFenetre+1)
	ecarts := make([]float64, 0, 2*hampelDemiFenetre+1)

	// Derniere vitesse jugee credible, point de depart du calcul d'acceleration.
	retenue := vitesses[0]

	for i := range vitesses {
		debut := max(0, i-hampelDemiFenetre)
		fin := min(len(vitesses), i+hampelDemiFenetre+1)

		fenetre = append(fenetre[:0], vitesses[debut:fin]...)
		mediane := medianeDe(fenetre)

		ecarts = ecarts[:0]
		for _, x := range fenetre {
			ecarts = append(ecarts, math.Abs(x-mediane))
		}

		// 1,4826 : facteur qui fait de la MAD un estimateur de l'ecart-type
		// pour une distribution normale.
		ecart := math.Max(medianeDe(ecarts)*1.4826, hampelPlancher)

		secondes := points[i].Duration.Seconds()

		aberrant := vitesses[i] > mediane+hampelSeuil*ecart
		impossible := vitesses[i]-retenue > accelerationMax*secondes

		if aberrant && impossible {
			distances[i] = mediane * secondes
			retenue = mediane
		} else {
			retenue = vitesses[i]
		}
	}

	return distances
}

// medianeDe trie la tranche fournie, qui doit donc etre un tampon de travail.
func medianeDe(valeurs []float64) float64 {
	if len(valeurs) == 0 {
		return 0
	}

	sort.Float64s(valeurs)

	milieu := len(valeurs) / 2
	if len(valeurs)%2 == 1 {
		return valeurs[milieu]
	}

	return (valeurs[milieu-1] + valeurs[milieu]) / 2
}

// VitessesLissees renvoie la vitesse de chaque point en m/s : sauts de position
// ecartes, puis moyenne sur une fenetre centree d'au moins cinq secondes.
//
// C'est cette serie qui alimente la courbe de vitesse d'une seance.
func VitessesLissees(points []MapPoint) []float64 {
	vitesses := make([]float64, len(points))
	if len(points) < 2 {
		return vitesses
	}

	distances := distancesFiltrees(points)

	// Sommes cumulees : la fenetre se lit alors par une soustraction, quelle que
	// soit sa largeur.
	cumulDistance := make([]float64, len(points))
	cumulSecondes := make([]float64, len(points))

	for i := 1; i < len(points); i++ {
		cumulDistance[i] = cumulDistance[i-1] + distances[i]
		cumulSecondes[i] = cumulSecondes[i-1] + math.Max(points[i].Duration.Seconds(), 0)
	}

	cible := fenetreLissage.Seconds()

	for i := range points {
		// La fenetre s'ouvre de part et d'autre du point, du cote le moins
		// fourni, jusqu'a couvrir la duree voulue ou buter sur un bord. Elle
		// reste donc centree tant que la trace le permet, et ne s'etire d'un
		// seul cote qu'au debut et a la fin.
		gauche, droite := i, i

		for cumulSecondes[droite]-cumulSecondes[gauche] < cible {
			gauchePossible := gauche > 0
			droitePossible := droite < len(points)-1

			if !gauchePossible && !droitePossible {
				break
			}

			versGauche := gauchePossible &&
				(!droitePossible ||
					cumulSecondes[i]-cumulSecondes[gauche] <= cumulSecondes[droite]-cumulSecondes[i])

			if versGauche {
				gauche--
			} else {
				droite++
			}
		}

		if duree := cumulSecondes[droite] - cumulSecondes[gauche]; duree > 0 {
			vitesses[i] = (cumulDistance[droite] - cumulDistance[gauche]) / duree
		}
	}

	return vitesses
}

// VitesseMaximale renvoie la plus haute vitesse soutenue de la trace, en m/s.
func VitesseMaximale(points []MapPoint) float64 {
	maxi := 0.0

	for _, v := range VitessesLissees(points) {
		if v > maxi && !math.IsNaN(v) && !math.IsInf(v, 0) {
			maxi = v
		}
	}

	return maxi
}

// VitesseMaximaleAvecAppareil renvoie la vitesse maximale d'une trace en tenant
// compte, si elle existe, de la vitesse rapportee par l'appareil.
func VitesseMaximaleAvecAppareil(points []MapPoint) float64 {
	return max(VitesseMaximale(points), vitesseMaximaleAppareil(points))
}

// vitesseMaximaleAppareil renvoie la plus haute vitesse rapportee par l'appareil
// lui-meme, quand la trace en porte une.
//
// Un compteur de velo ou une montre mesurent la vitesse autrement que par la
// position, et generalement mieux ; la serie passe tout de meme par le filtre,
// rien ne garantit qu'elle ne soit pas elle-meme derivee du GPS.
func vitesseMaximaleAppareil(points []MapPoint) float64 {
	releves := make([]float64, 0, len(points))
	for i := range points {
		if v, ok := points[i].ExtraMetrics["speed"]; ok && v > 0 {
			releves = append(releves, v)
		}
	}

	if len(releves) < 2*hampelDemiFenetre {
		return 0
	}

	// La serie est reinjectee comme des intervalles d'une seconde pour reutiliser
	// le meme filtre, puis la fenetre de lissage.
	pseudo := make([]MapPoint, len(releves))
	for i, v := range releves {
		pseudo[i] = MapPoint{Distance: v, Duration: time.Second}
	}

	return VitesseMaximale(pseudo)
}
