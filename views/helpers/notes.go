package helpers

import "regexp"

// Les notes sont du Markdown : une image y est ecrite ![alt](url), eventuellement
// enveloppee dans un lien. On ne retient que l'url de l'image.
var imageMarkdown = regexp.MustCompile(`!\[[^\]]*\]\(\s*([^)\s]+)`)

// ImagesInNotes renvoie les url des images citees dans des notes, au plus max.
//
// Sert a montrer un apercu des photos d'une seance dans la liste, ou les notes
// completes ne sont pas rendues.
func ImagesInNotes(notes string, max int) []string {
	if notes == "" {
		return nil
	}

	trouvees := imageMarkdown.FindAllStringSubmatch(notes, max)
	if trouvees == nil {
		return nil
	}

	urls := make([]string, 0, len(trouvees))
	for _, t := range trouvees {
		urls = append(urls, t[1])
	}

	return urls
}

// CountImagesInNotes compte toutes les images citees dans des notes.
func CountImagesInNotes(notes string) int {
	if notes == "" {
		return 0
	}

	return len(imageMarkdown.FindAllStringIndex(notes, -1))
}
