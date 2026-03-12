package engine

import "math"

const earthRadiusKm = 6371.0

// HaversineDistance returns the distance in km between two lat/lon points.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := degToRad(lat2 - lat1)
	dLon := degToRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// Centroid returns the geographic centroid of a set of lat/lon points.
func Centroid(lats, lons []float64) (float64, float64) {
	if len(lats) == 0 {
		return 0, 0
	}
	var x, y, z float64
	for i := range lats {
		latR := degToRad(lats[i])
		lonR := degToRad(lons[i])
		x += math.Cos(latR) * math.Cos(lonR)
		y += math.Cos(latR) * math.Sin(lonR)
		z += math.Sin(latR)
	}
	n := float64(len(lats))
	x /= n
	y /= n
	z /= n
	lon := math.Atan2(y, x)
	hyp := math.Sqrt(x*x + y*y)
	lat := math.Atan2(z, hyp)
	return radToDeg(lat), radToDeg(lon)
}

// ProjectPosition projects a position forward along a great-circle path.
// heading in degrees, distanceKm in km.
func ProjectPosition(lat, lon, heading, distanceKm float64) (float64, float64) {
	headingRad := degToRad(heading)
	latRad := degToRad(lat)
	lonRad := degToRad(lon)
	angularDist := distanceKm / earthRadiusKm

	newLat := math.Asin(math.Sin(latRad)*math.Cos(angularDist) +
		math.Cos(latRad)*math.Sin(angularDist)*math.Cos(headingRad))
	newLon := lonRad + math.Atan2(
		math.Sin(headingRad)*math.Sin(angularDist)*math.Cos(latRad),
		math.Cos(angularDist)-math.Sin(latRad)*math.Sin(newLat),
	)
	return radToDeg(newLat), radToDeg(newLon)
}

func degToRad(d float64) float64 { return d * math.Pi / 180.0 }
func radToDeg(r float64) float64 { return r * 180.0 / math.Pi }

// MaxRadius returns the maximum distance from a centroid to any point.
func MaxRadius(centLat, centLon float64, lats, lons []float64) float64 {
	maxR := 0.0
	for i := range lats {
		d := HaversineDistance(centLat, centLon, lats[i], lons[i])
		if d > maxR {
			maxR = d
		}
	}
	return maxR
}
