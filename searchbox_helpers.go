package mapbox

import (
	"strconv"
	"strings"
)

func formatCoordParam(lng, lat float64) string {
	return strconv.FormatFloat(lng, 'f', -1, 64) + "," + strconv.FormatFloat(lat, 'f', -1, 64)
}

func formatBBoxParam(bb *BoundingBox) string {
	return strings.Join([]string{
		strconv.FormatFloat(bb.MinLongitude, 'f', -1, 64),
		strconv.FormatFloat(bb.MinLatitude, 'f', -1, 64),
		strconv.FormatFloat(bb.MaxLongitude, 'f', -1, 64),
		strconv.FormatFloat(bb.MaxLatitude, 'f', -1, 64),
	}, ",")
}
