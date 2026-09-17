package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/spf13/cobra"
)

// recalculateSpeedsCmd remet a jour la vitesse maximale enregistree de chaque
// seance a partir des points deja en base.
//
// Les seances importees avant l'ajout du filtre de vitesse portent une valeur
// prise sur un saut GPS -- jusqu'a 9 800 km/h sur cet export. La page d'une
// seance recalcule desormais la valeur a l'affichage, mais les classements et
// les records lisent la colonne map_data.max_speed : il faut donc la reecrire
// une fois.
//
// A lancer conteneur arrete, l'application gardant la base pour elle :
//
//	docker run --rm -v <volume>:/data -e WT_DSN=/data/database.db \
//	  --entrypoint /app/wt-debug <image> workouts recalculate-speeds
//
// Les points sont relus seance par seance, en SQL brut : le cache de l'ORM
// (pkg/database/gorm_cache.go) ne connait ni plafond ni eviction, y passer une
// requete sur les 2 Go de points garderait toute la base en memoire.
func (c *cli) recalculateSpeedsCmd() *cobra.Command {
	var simulation bool

	cmd := &cobra.Command{
		Use:   "recalculate-speeds",
		Short: "Recompute the stored maximum speed of every workout from its GPS points",
		RunE: func(cmd *cobra.Command, args []string) error {
			db := c.getDatabase()

			var ids []uint64
			if err := db.Raw("SELECT id FROM map_data ORDER BY id").Scan(&ids).Error; err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "%d seances a examiner\n", len(ids))

			var corrigees int

			debut := time.Now()

			for i, id := range ids {
				ancienne, nouvelle, ok, err := recalculateSpeedFor(c, id)
				if err != nil {
					return err
				}

				if !ok {
					continue
				}

				// En dessous, la difference ne change aucun affichage.
				if math.Abs(nouvelle-ancienne) < 0.01 {
					continue
				}

				corrigees++

				fmt.Printf("map_data %-6d %8.1f -> %6.1f km/h\n", id, ancienne*3.6, nouvelle*3.6)

				if !simulation {
					if err := db.Exec("UPDATE map_data SET max_speed = ? WHERE id = ?", nouvelle, id).Error; err != nil {
						return err
					}
				}

				if (i+1)%100 == 0 {
					fmt.Fprintf(os.Stderr, "  %d/%d, %s ecoulees\n", i+1, len(ids), time.Since(debut).Round(time.Second))
				}
			}

			mot := "corrigees"
			if simulation {
				mot = "a corriger (simulation)"
			}

			fmt.Fprintf(os.Stderr, "%d seances %s en %s\n", corrigees, mot, time.Since(debut).Round(time.Second))

			return nil
		},
	}

	cmd.Flags().BoolVar(&simulation, "dry-run", false, "Report what would change without writing")

	return cmd
}

// recalculateSpeedFor relit les points d'une seule seance et renvoie l'ancienne
// et la nouvelle vitesse maximale, en m/s.
func recalculateSpeedFor(c *cli, mapDataID uint64) (float64, float64, bool, error) {
	var ligne struct {
		Points   []byte
		MaxSpeed float64
	}

	err := c.getDatabase().Raw(
		"SELECT md.max_speed, mdd.points FROM map_data md "+
			"JOIN map_data_details mdd ON mdd.map_data_id = md.id WHERE md.id = ?",
		mapDataID,
	).Row().Scan(&ligne.MaxSpeed, &ligne.Points)
	if err != nil {
		// Une seance sans points (saisie a la main) n'a rien a recalculer.
		return 0, 0, false, nil //nolint:nilerr
	}

	var points []database.MapPoint
	if err := json.Unmarshal(ligne.Points, &points); err != nil {
		return 0, 0, false, err
	}

	if len(points) < 2 {
		return 0, 0, false, nil
	}

	return ligne.MaxSpeed, database.VitesseMaximaleAvecAppareil(points), true, nil
}
