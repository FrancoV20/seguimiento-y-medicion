package web

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-10 (T027): textos, formato y modelo de vista de "Calidad y avance". Los textos de la pantalla
// se definen solo acá (FR-019).

const (
	TitleQuality           = "Calidad y avance"
	MsgQualityNoSprints    = "Aún no hay Sprints finalizados"
	MsgQualityActiveHint   = "El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo"
	MsgQualityNotAvailable = "No disponible"

	msgQualityWithoutPointsOne  = "1 historia sin Story Points no se suma a los planificados"
	msgQualityWithoutPointsMany = "%d historias sin Story Points no se suman a los planificados"
	msgQualityStories           = "%d de %d historias completadas"

	pathQuality = "/calidad"
)

// qualityPageData es lo que recibe templates/quality.html.
type qualityPageData struct {
	Sprints    []qualitySprintLink
	ActiveHint string       // MsgQualityActiveHint con el número, o ""
	NoSprints  string       // MsgQualityNoSprints, o ""
	Selected   *qualityView // nil si no hay Sprints finalizados
}

// qualitySprintLink es un enlace de la lista de Sprints finalizados.
type qualitySprintLink struct {
	Number  int
	URL     string // "/calidad?sprint=ID"
	Current bool
}

// qualityView son los números ya formateados del Sprint elegido.
type qualityView struct {
	SprintNumber    int
	StoriesText     string // "3 de 4 historias completadas"
	Percentage      string // "75,00 %" o "No disponible"
	PointsPlanned   int
	PointsCompleted int
	PointsWarning   string // "" si no hay historias sin Story Points
	DefectsDetected int
	DefectsResolved int
}

// qualityPercentText pasa "75.00" a "75,00 %"; "No disponible" queda igual.
func qualityPercentText(percentage string) string {
	if percentage == MsgQualityNotAvailable {
		return percentage
	}
	return strings.Replace(percentage, ".", ",", 1) + " %"
}

// qualityWithoutPointsWarning devuelve la advertencia de historias sin Story Points, en singular o
// en plural, o "" si no hay ninguna.
func qualityWithoutPointsWarning(n int) string {
	switch {
	case n == 1:
		return msgQualityWithoutPointsOne
	case n > 1:
		return fmt.Sprintf(msgQualityWithoutPointsMany, n)
	default:
		return ""
	}
}

// selectQualitySprint elige el Sprint pedido en la dirección si está entre los finalizados del
// proyecto; si no (vacío, inválido, activo, inexistente o ajeno), el más reciente. Devuelve false si
// no hay Sprints finalizados.
func selectQualitySprint(raw string, list QualitySprintList) (QualitySprintOption, bool) {
	if len(list.Finished) == 0 {
		return QualitySprintOption{}, false
	}
	if id := parseID(raw); id != 0 {
		for _, option := range list.Finished {
			if option.ID == id {
				return option, true
			}
		}
	}
	return list.Finished[0], true
}

// qualityActiveHint devuelve el aviso del Sprint activo, o "" si no hay uno.
func qualityActiveHint(list QualitySprintList) string {
	if list.ActiveNumber == 0 {
		return ""
	}
	return fmt.Sprintf(MsgQualityActiveHint, list.ActiveNumber)
}

// newQualityPageData arma la pantalla con la lista, el Sprint elegido y sus números.
func newQualityPageData(list QualitySprintList, chosen QualitySprintOption, quality metrics.SprintQuality) qualityPageData {
	data := qualityPageData{
		Sprints:    make([]qualitySprintLink, 0, len(list.Finished)),
		ActiveHint: qualityActiveHint(list),
		Selected: &qualityView{
			SprintNumber:    chosen.Number,
			StoriesText:     fmt.Sprintf(msgQualityStories, quality.StoriesCompleted, quality.StoriesPlanned),
			Percentage:      qualityPercentText(quality.CompletionPercentage),
			PointsPlanned:   quality.PointsPlanned,
			PointsCompleted: quality.PointsCompleted,
			PointsWarning:   qualityWithoutPointsWarning(quality.StoriesWithoutPoints),
			DefectsDetected: quality.DefectsDetected,
			DefectsResolved: quality.DefectsResolved,
		},
	}
	for _, option := range list.Finished {
		data.Sprints = append(data.Sprints, qualitySprintLink{
			Number:  option.Number,
			URL:     pathQuality + "?sprint=" + strconv.FormatInt(option.ID, 10),
			Current: option.ID == chosen.ID,
		})
	}
	return data
}

// newQualityEmptyPageData arma la pantalla sin Sprints finalizados.
func newQualityEmptyPageData(list QualitySprintList) qualityPageData {
	return qualityPageData{ActiveHint: qualityActiveHint(list), NoSprints: MsgQualityNoSprints}
}
