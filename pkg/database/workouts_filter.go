package database

import (
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// Nombre de seances par page. Le carnet de cette installation en compte pres
// d'un millier : tout afficher d'un coup demandait autant de vignettes de trace
// au serveur, et rendait le defilement inutilisable.
const WorkoutsParPage = 24

type WorkoutFilters struct {
	db       *gorm.DB
	Type     WorkoutType `query:"type"`
	Since    string      `query:"since"`
	OrderBy  string      `query:"order_by"`
	OrderDir string      `query:"order_dir"`
	Year     string      `query:"year"`
	Page     int         `query:"page"`
	Active   bool        `query:"active"`
}

func GetWorkoutsFilters(c *echo.Context) (*WorkoutFilters, error) {
	filters := WorkoutFilters{}

	if err := c.Bind(&filters); err != nil {
		return nil, err
	}

	filters.setDefaults()

	return &filters, nil
}

func (wf *WorkoutFilters) setDefaults() {
	// Un carnet importe depuis un export Strava commence souvent dix ans plus
	// tot : une periode par defaut masquerait la moitie des seances sans le
	// dire.
	if wf.Since == "" {
		wf.Since = "forever"
	}

	if wf.OrderBy == "" {
		wf.OrderBy = "date"
	}

	if wf.OrderDir == "" {
		wf.OrderDir = "desc"
	}

	if wf.Page < 1 {
		wf.Page = 1
	}
}

// EstFiltree indique si un critere autre que l'ordre par defaut est pose.
func (wf *WorkoutFilters) EstFiltree() bool {
	return wf.Type != "" || wf.Year != "" || (wf.Since != "" && wf.Since != "forever")
}

// Where n'applique que les criteres de selection : c'est la requete a compter.
func (wf *WorkoutFilters) Where(db *gorm.DB) *gorm.DB {
	wf.db = db

	wf.setTypeFilter()
	wf.setYearFilter()
	wf.setSinceFilter()

	return wf.db
}

// ToQuery ajoute le tri aux criteres de selection.
func (wf *WorkoutFilters) ToQuery(db *gorm.DB) *gorm.DB {
	wf.db = wf.Where(db)

	wf.setOrderFilter()

	return wf.db
}

// Paginate borne la requete a la page demandee.
func (wf *WorkoutFilters) Paginate(db *gorm.DB) *gorm.DB {
	return db.Limit(WorkoutsParPage).Offset((wf.Page - 1) * WorkoutsParPage)
}

// NombreDePages renvoie le nombre de pages pour un total de seances donne.
func NombreDePages(total int64) int {
	pages := int((total + WorkoutsParPage - 1) / WorkoutsParPage)
	if pages < 1 {
		return 1
	}

	return pages
}

func (wf *WorkoutFilters) setTypeFilter() {
	if wf.Type == "" {
		return
	}

	wf.db = wf.db.Where(&Workout{Type: wf.Type})
}

func (wf *WorkoutFilters) setYearFilter() {
	if wf.Year == "" {
		return
	}

	wf.db = wf.db.Where(yearExpression(wf.db.Name())+" = ?", wf.Year)
}

func (wf *WorkoutFilters) setSinceFilter() {
	if wf.Since == "" || wf.Since == "forever" {
		return
	}

	sqlDialect := wf.db.Name()
	wf.db = wf.db.Where(GetDateLimitExpression(sqlDialect), "-"+wf.Since)
}

func (wf *WorkoutFilters) setOrderFilter() {
	if wf.OrderBy == "" {
		return
	}

	wf.db = wf.db.Select("workouts.*").Joins("left join map_data on workouts.id = map_data.workout_id")

	dir := wf.OrderDir
	if dir == "" {
		dir = "asc"
	}

	switch wf.OrderBy {
	case "date":
		wf.db = wf.db.Order("workouts." + wf.OrderBy + " " + dir)
	case "total_distance", "total_duration", "total_weight", "total_repetitions", "total_up", "total_down",
		"average_speed_no_pause", "max_speed":
		wf.db = wf.db.Order("map_data." + wf.OrderBy + " " + dir)
	}
}
