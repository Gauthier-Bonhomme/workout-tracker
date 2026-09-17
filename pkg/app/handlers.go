package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/invopop/ctxi18n/i18n"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/views/partials"
	"github.com/jovandeginste/workout-tracker/v2/views/user"
	"github.com/labstack/echo/v5"
	"github.com/spf13/cast"
)

var ErrUserNotFound = errors.New("user not found")

func (a *App) redirectWithError(c *echo.Context, target string, err error) error {
	if err != nil {
		a.addErrorT(c, "alerts.something_wrong", i18n.M{"message": err.Error()})
	}

	return c.Redirect(http.StatusFound, target)
}

func (a *App) statisticsHandler(c *echo.Context) error {
	u := a.getCurrentUser(c)
	if u.IsAnonymous() {
		return a.redirectWithError(c, a.Reverse("user-signout"), ErrUserNotFound)
	}

	statisticsParams := struct {
		Since string `query:"since"`
		Per   string `query:"per"`
	}{
		Since: "1 year",
		Per:   "month",
	}

	if err := c.Bind(&statisticsParams); err != nil {
		return a.redirectWithError(c, a.Reverse("dashboard"), err)
	}

	return Render(c, http.StatusOK, user.Statistics(u, statisticsParams.Since, statisticsParams.Per))
}

func (a *App) dailyDeleteHandler(c *echo.Context) error {
	u := a.getCurrentUser(c)
	d := c.Param("date")

	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	m, err := u.GetMeasurementForDate(t)
	if err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	if err := m.Delete(a.db); err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	if isHtmx(c) {
		c.Response().Header().Set("Hx-Redirect", a.Reverse("daily"))
		return c.String(http.StatusFound, "ok")
	}

	return c.Redirect(http.StatusFound, a.Reverse("daily"))
}

func (a *App) dailyUpdateHandler(c *echo.Context) error {
	d := &Measurement{units: a.getCurrentUser(c).PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	m, err := a.getCurrentUser(c).GetMeasurementForDate(d.Time())
	if err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	d.Update(m)

	if err := m.Save(a.db); err != nil {
		return a.redirectWithError(c, a.Reverse("daily"), err)
	}

	return c.Redirect(http.StatusFound, a.Reverse("daily"))
}

func (a *App) dailyHandler(c *echo.Context) error {
	u := a.getCurrentUser(c)

	count := 20
	if cs := c.QueryParam("count"); cs != "" {
		if ci, err := cast.ToIntE(cs); err == nil {
			count = ci
		} else {
			return a.redirectWithError(c, a.Reverse("daily"), err)
		}
	}

	return Render(c, http.StatusOK, user.Daily(u, count))
}

// Nombre de seances presentees en pied de tableau de bord. Au-dela, la page
// double la liste des seances sans rien apprendre de plus.
const dernieresSeances = 5

// donneesTableauDeBord rassemble les totaux d'un carnet.
//
// La page ne chargeait pas seulement ces totaux : elle demandait toutes les
// seances de l'utilisateur pour les passer a un calendrier qui, lui, allait les
// chercher par son API. Un millier de lignes lues a chaque affichage, pour
// rien.
func (a *App) donneesTableauDeBord(u *database.User) (user.DonneesTableauDeBord, error) {
	tableau := user.DonneesTableauDeBord{User: u}

	var err error

	if tableau.Resume, err = u.GetResume(); err != nil {
		return tableau, err
	}

	if tableau.Annees, err = u.GetResumeParAnnee(); err != nil {
		return tableau, err
	}

	if tableau.Types, err = u.GetResumeParType(); err != nil {
		return tableau, err
	}

	if tableau.Dernieres, err = u.GetWorkouts(a.db.Limit(dernieresSeances)); err != nil {
		return tableau, err
	}

	return tableau, nil
}

func (a *App) dashboardHandler(c *echo.Context) error {
	u := a.getCurrentUser(c)
	if u.IsAnonymous() {
		return a.redirectWithError(c, a.Reverse("user-signout"), ErrUserNotFound)
	}

	tableau, err := a.donneesTableauDeBord(u)
	if err != nil {
		return a.redirectWithError(c, a.Reverse("user-signout"), err)
	}

	if tableau.Autres, err = database.GetUsers(a.db); err != nil {
		return a.redirectWithError(c, a.Reverse("user-signout"), err)
	}

	return Render(c, http.StatusOK, user.Show(tableau))
}

func (a *App) userLoginHandler(c *echo.Context) error {
	return Render(c, http.StatusOK, user.Login())
}

func (a *App) lookupAddressHandler(c *echo.Context) error {
	q := c.FormValue("location")

	results, err := geocoder.Search(q)
	if err != nil {
		a.addErrorT(c, "alerts.something_wrong", i18n.M{"message": err.Error()})
	}

	return Render(c, http.StatusOK, partials.AddressResults(results))
}

func (a *App) heatmapHandler(c *echo.Context) error {
	u := a.getCurrentUser(c)
	if u.IsAnonymous() {
		return a.redirectWithError(c, a.Reverse("user-signout"), ErrUserNotFound)
	}

	w, err := u.GetWorkouts(a.db)
	if err != nil {
		return a.redirectWithError(c, a.Reverse("user-signout"), err)
	}

	return Render(c, http.StatusOK, user.Heatmap(w))
}

func Render(ctx *echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
