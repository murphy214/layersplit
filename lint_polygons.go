package layersplit 

import (
	pc "github.com/murphy214/polyclip"
	//m "github.com/murphy214/mercantile"
	"github.com/paulmach/go.geojson"
)


// Lints properties within multigeometies
func Lint_Properties(props map[string]interface{}) map[string]interface{} {
	newprops := map[string]interface{}{}
	for k,v := range props {
		if k != "id" {
			newprops[k] = v
		}
	}
	return newprops
}

// Splits multiple geometries into single geoemetries
func Split_Multi(gjson *geojson.FeatureCollection) *geojson.FeatureCollection {
	// splitting multi geometriess
	c := make(chan []*geojson.Feature)
	for _,i := range gjson.Features {
		go func(i *geojson.Feature,c chan []*geojson.Feature) {
			if i.Geometry.Type == "MultiLineString" {
				props := i.Properties
				props = Lint_Properties(props)
				newfeats := []*geojson.Feature{}
				for _,newline := range i.Geometry.MultiLineString {
					newfeats = append(newfeats,&geojson.Feature{Geometry:&geojson.Geometry{LineString:newline,Type:"LineString"},Properties:props})
				}

				c <- newfeats
			} else if i.Geometry.Type == "MultiPolygon" {
				props := i.Properties
				props = Lint_Properties(props)

				newfeats := []*geojson.Feature{}
				for _,newline := range i.Geometry.MultiPolygon {
					newfeats = append(newfeats,&geojson.Feature{Geometry:&geojson.Geometry{Polygon:newline,Type:"Polygon"},Properties:props})
				}
				c <- newfeats
			} else if i.Geometry.Type == "MultiPoint" {
				props := i.Properties

				props = Lint_Properties(props)

				newfeats := []*geojson.Feature{}
				for _,newline := range i.Geometry.MultiPoint {
					newfeats = append(newfeats,&geojson.Feature{Geometry:&geojson.Geometry{Point:newline,Type:"Point"},Properties:props})
				}
				c <- newfeats
			} else {
				i.Properties = Lint_Properties(i.Properties)

				c <- []*geojson.Feature{i}
			}
		}(i,c)
	}
	newfeats := []*geojson.Feature{}
	for range gjson.Features {
		newfeats = append(newfeats,<-c...)
	}	
	return &geojson.FeatureCollection{Features:newfeats}
} 


// Overlaps returns whether r1 and r2 have a non-empty intersection.
func Within(big pc.Rectangle, small pc.Rectangle) bool {
	return (big.Min.X <= small.Min.X) && (big.Max.X >= small.Max.X) &&
		(big.Min.Y <= small.Min.Y) && (big.Max.Y >= small.Max.Y)
}

// a check to see if each point of a contour is within the bigger
func WithinAll(big pc.Contour, small pc.Contour) bool {
	totalbool := true
	for _, pt := range small {
		boolval := big.Contains(pt)
		if boolval == false {
			totalbool = false
		}
	}
	return totalbool
}

// creating a list with all of the intersecting contours
// this function returns a list of all the constituent contours as well as
// a list of their keys
func Sweep_Contmap(bb pc.Rectangle, intcont pc.Contour, contmap map[int]pc.Contour) []int {
	newlist := []int{}
	for k, v := range contmap {
		// getting the bounding box
		bbtest := v.BoundingBox()

		// getting within bool
		withinbool := Within(bb, bbtest)

		// logic for if within bool is true
		if withinbool == true {
			withinbool = WithinAll(intcont, v)
		}

		// logic for when we know the contour is within the polygon
		if withinbool == true {
			newlist = append(newlist, k)
		}
	}
	return newlist
}

// getting the outer keys of contours that will be turned into polygons
func make_polygon_list(totalkeys []int, contmap map[int]pc.Contour, relationmap map[int][]int) []pc.Polygon {
	keymap := map[int]string{}
	for _, i := range totalkeys {
		keymap[i] = ""
	}

	// making polygon map
	polygonlist := []pc.Polygon{}
	for k, v := range contmap {
		_, ok := keymap[k]
		if ok == false {
			newpolygon := pc.Polygon{v}
			otherconts := relationmap[k]
			for _, cont := range otherconts {
				newpolygon.Add(contmap[cont])
			}

			// finally adding to list
			polygonlist = append(polygonlist, newpolygon)
		}
	}
	return polygonlist

}

// creates a within map or a mapping of each edge
func Create_Withinmap(contmap map[int]pc.Contour) []pc.Polygon {
	totalkeys := []int{}
	relationmap := map[int][]int{}
	for k, v := range contmap {
		bb := v.BoundingBox()
		keys := Sweep_Contmap(bb, v, contmap)
		relationmap[k] = keys
		totalkeys = append(totalkeys, keys...)
	}

	return make_polygon_list(totalkeys, contmap, relationmap)
}

// lints each polygon
// takes abstract polygon rings that may contain polygon rings
// and returns geojson arranged polygon sets
func Lint_Polygons(polygon pc.Polygon) []pc.Polygon {
	if len(polygon) == 1 {
		return []pc.Polygon{polygon}
	}
	contmap := map[int]pc.Contour{}
	for i, cont := range polygon {
		contmap[i] = cont
	}
	return Create_Withinmap(contmap)

}

// from a pc.Polygon representation (clipping representation)
// to a [][][]float64 representation
func Convert_Float(poly pc.Polygon) [][][]float64 {
	total := [][][]float64{}
	for _, cont := range poly {
		contfloat := [][]float64{}
		for _, pt := range cont {
			contfloat = append(contfloat, []float64{pt.X, pt.Y})
		}
		total = append(total, contfloat)
	}
	return total
}



