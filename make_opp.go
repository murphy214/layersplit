package layersplit

import (
	m "github.com/murphy214/mercantile"
	pc "github.com/murphy214/polyclip"
	"math"
)

func Area_Contour(cont pc.Contour) float64 {
	total := 0.0
	var firstpt, oldpt pc.Point
	firstpt = cont[0]
	for i, pt := range cont {
		if i != 0 {
			total += (oldpt.X * pt.Y) - (oldpt.Y * pt.X)
		}
		oldpt = pt
	}

	total += (oldpt.X * firstpt.Y) - (oldpt.Y * firstpt.X)
	return math.Abs(total) / 2.0
}
func Area_Polygon(polygon pc.Polygon) (float64, pc.Polygon) {
	total := 0.0
	newpolygon := pc.Polygon{}
	for _, cont := range polygon {
		//total += Area_Contour(cont)
		area := Area_Contour(cont)
		total += area
		if area > .0001 {
			newpolygon.Add(cont)
		}
	}
	return total, newpolygon
}

func Lint_Polygon_Prec(polygon pc.Polygon) pc.Polygon {
	newpolygon := pc.Polygon{}
	for _, cont := range polygon {
		newcont := pc.Contour{}
		for _, pt := range cont {
			newcont.Add(pc.Point{X: Round(pt.X, .5, 6), Y: Round(pt.Y, .5, 6)})
		}
		newpolygon.Add(newcont)
	}
	return newpolygon
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func Make_Opp(ztt []Polygon, layername string) []Polygon {

	totalpolygon := pc.Polygon{}
	for _, row := range ztt {
		for _, cont := range row.Polygon {
			totalpolygon.Add(cont)
		}
	}

	bd := totalpolygon.BoundingBox()
	bds := m.Extrema{W: bd.Min.X, E: bd.Max.X, S: bd.Min.Y, N: bd.Max.Y}
	// ne,nw,sw,se
	bigpoly := pc.Polygon{{{bds.E, bds.N}}, {{bds.W, bds.N}}, {{bds.W, bds.S}}, {{bds.E, bds.S}}}

	results := bigpoly.Construct(pc.XOR, totalpolygon)
	_, results = Area_Polygon(results)

	for _, i := range results {
		bd := i.BoundingBox()
		bds := m.Extrema{W: bd.Min.X, E: bd.Max.X, S: bd.Min.Y, N: bd.Max.Y}

		ztt = append(ztt, Polygon{Polygon: Lint_Polygon_Prec(pc.Polygon{i}), Area: "NONE", Layer: layername, Bounds: bds})
	}
	return ztt
}
