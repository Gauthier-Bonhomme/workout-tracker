package database

import (
	"time"

	"gorm.io/gorm"
)

// Resume rassemble les totaux d'un carnet, tous sports confondus.
//
// GetTotals ne compte qu'un seul type de seance, celui choisi dans le profil :
// sur un carnet qui melange velo, course et randonnee, le chiffre d'ouverture
// du tableau de bord n'avait alors aucun rapport avec ce que la liste montre.
type Resume struct {
	Duree    time.Duration
	Distance float64
	Up       float64
	Seances  int
}

// ResumeAnnee est la ligne d'une annee dans le recapitulatif.
type ResumeAnnee struct {
	Annee    string
	Duree    time.Duration
	Distance float64
	Seances  int
}

// ResumeType est la ligne d'un sport dans le recapitulatif.
type ResumeType struct {
	Type     WorkoutType
	Duree    time.Duration
	Distance float64
	Seances  int
}

// yearExpression renvoie l'extraction de l'annee d'une seance, en texte.
func yearExpression(sqlDialect string) string {
	if sqlDialect == postgresDialect {
		return "to_char(workouts.date, 'YYYY')"
	}

	return "strftime('%Y', workouts.date)"
}

// resumeQuery part des seances et non des traces : une seance saisie a la main
// n'a pas de ligne dans map_data, une jointure interne la ferait disparaitre du
// compte.
func (u *User) resumeQuery() *gorm.DB {
	return u.db.
		Table("workouts").
		Joins("left join map_data on workouts.id = map_data.workout_id").
		Where("workouts.user_id = ?", u.ID)
}

var resumeColonnes = []string{
	"count(*) as seances",
	"coalesce(sum(map_data.total_distance), 0) as distance",
	"coalesce(sum(map_data.total_up), 0) as up",
	"coalesce(sum(map_data.total_duration), 0) as duree",
}

// GetResume renvoie les totaux du carnet entier.
func (u *User) GetResume() (*Resume, error) {
	if u.IsAnonymous() {
		return nil, ErrAnonymousUser
	}

	r := &Resume{}

	if err := u.resumeQuery().Select(resumeColonnes).Scan(r).Error; err != nil {
		return nil, err
	}

	return r, nil
}

// GetResumeParAnnee renvoie une ligne par annee, dans l'ordre chronologique.
// C'est l'ossature de la navigation dans un carnet qui court sur plusieurs
// annees.
func (u *User) GetResumeParAnnee() ([]ResumeAnnee, error) {
	if u.IsAnonymous() {
		return nil, ErrAnonymousUser
	}

	annee := yearExpression(u.db.Dialector.Name())

	var lignes []ResumeAnnee

	err := u.resumeQuery().
		Select(annee+" as annee",
			"count(*) as seances",
			"coalesce(sum(map_data.total_distance), 0) as distance",
			"coalesce(sum(map_data.total_duration), 0) as duree").
		Group(annee).
		Order("annee asc").
		Scan(&lignes).Error
	if err != nil {
		return nil, err
	}

	return lignes, nil
}

// GetResumeParType renvoie une ligne par sport, du plus pratique au moins
// pratique.
func (u *User) GetResumeParType() ([]ResumeType, error) {
	if u.IsAnonymous() {
		return nil, ErrAnonymousUser
	}

	var lignes []ResumeType

	err := u.resumeQuery().
		Select("workouts.type as type",
			"count(*) as seances",
			"coalesce(sum(map_data.total_distance), 0) as distance",
			"coalesce(sum(map_data.total_duration), 0) as duree").
		Group("workouts.type").
		Order("count(*) desc").
		Scan(&lignes).Error
	if err != nil {
		return nil, err
	}

	return lignes, nil
}
