package layersplit

import (
	"encoding/json"
	"fmt"
	m "github.com/murphy214/mercantile"
	"github.com/murphy214/polyclip"
	"math/rand"
	"strings"
	"time"
)

// Point represents a point in space.
type Polygon struct {
	Polygon    polyclip.Polygon
	Bounds     m.Extrema
	Layer      string
	Area       string
	Polystring string
	Layers     map[string]string
	Pos        int
	Properties []interface{}
}

// Point represents a point in space.
type Output_Struct struct {
	Polylist   []Polygon
	Polystring string
}

type ResponseCoords2 struct {
	Coords [][][]float64 `json:"coords"`
}

// gets the coordstring into a slice the easiest way I'm aware of
func get_coords_json2(stringcoords string) [][][]float64 {
	stringcoords = fmt.Sprintf(`{"coords":%s}`, stringcoords)
	res := ResponseCoords2{}
	json.Unmarshal([]byte(stringcoords), &res)

	return res.Coords
}

// function for getting the extrema of an alignment
func get_extrema_coords(coords [][]float64) m.Extrema {
	north := -1000.
	south := 1000.
	east := -1000.
	west := 1000.
	lat := 0.
	long := 0.
	for i := range coords {
		if len(coords[i]) == 2 {
			lat = coords[i][1]
			long = coords[i][0]

			if lat > north {
				north = lat
			}
			if lat < south {
				south = lat
			}
			if long > east {
				east = long
			}
			if long < west {
				west = long
			}
			//fmt.Print(long, lat, "\n")
		}
	}

	// sorting both lats and longs
	return m.Extrema{S: south, W: west, N: north, E: east}

}
func make_polygon(coords [][][]float64) polyclip.Polygon {
	thing2 := polyclip.Contour{}
	things := polyclip.Polygon{}
	for _, coord := range coords {
		thing2 = polyclip.Contour{}

		for _, i := range coord {
			if len(i) == 2 {
				// moving sign in 10 ** -7 pla
				sign1 := rand.Intn(1)
				factor1 := float64(rand.Intn(100)) * .00000001
				if sign1 == 1 {
					factor1 = factor1 * -1
				}
				sign2 := rand.Intn(1)
				factor2 := float64(rand.Intn(100)) * .00000001
				if sign2 == 1 {
					factor2 = factor2 * -1
				}
				thing2.Add(polyclip.Point{X: i[0] + factor1, Y: i[1] + factor2})
			}
		}
		things.Add(thing2)
	}

	return things
}

func Make_Layer(polys [][]string, layername string) []Polygon {
	layer := []Polygon{}

	c := make(chan []Polygon)
	for i, polyrow := range polys {
		polygonstrings := strings.Split(polyrow[1], "|")
		go func(polygonstrings []string, polyrow []string, c chan<- []Polygon) {
			templayer := []Polygon{}
			for _, polygonstring := range polygonstrings {

				polygon := get_coords_json2(polygonstring)
				polygonc := make_polygon(polygon)
				extrema := get_extrema_coords(polygon[0])
				templayer = append(templayer, Polygon{Layer: layername, Polygon: polygonc, Bounds: extrema, Polystring: polygonstring, Area: string(polyrow[0]), Pos: i})

			}
			c <- templayer
		}(polygonstrings, polyrow, c)

	}
	for ii := 0; ii < len(polys); ii++ {
		select {
		case msg1 := <-c:
			layer = append(layer, msg1...)

		}
	}

	return layer
}

func Make_Layer_Properties(polys [][]string, layername string, props [][]interface{}) []Polygon {
	layer := []Polygon{}

	c := make(chan []Polygon)
	for i, polyrow := range polys {
		prop := props[i]
		polygonstrings := strings.Split(polyrow[1], "|")
		go func(polygonstrings []string, polyrow []string, prop []interface{}, c chan<- []Polygon) {
			templayer := []Polygon{}
			for _, polygonstring := range polygonstrings {

				polygon := get_coords_json2(polygonstring)
				polygonc := make_polygon(polygon)
				extrema := get_extrema_coords(polygon[0])
				templayer = append(templayer, Polygon{Layer: layername, Polygon: polygonc, Bounds: extrema, Polystring: polygonstring, Area: string(polyrow[0]), Pos: i, Properties: prop})

			}
			c <- templayer
		}(polygonstrings, polyrow, prop, c)

	}
	for ii := 0; ii < len(polys); ii++ {
		select {
		case msg1 := <-c:
			layer = append(layer, msg1...)

		}
	}

	return layer
}
func check(poly1 Polygon, poly2 Polygon, typeval string, ch chan<- bool) {
	if typeval == "INTERSECTION" {
		poly1.Polygon.Construct(polyclip.INTERSECTION, poly2.Polygon)

	}
	if typeval == "DIFFERENCE" {
		poly1.Polygon.Construct(polyclip.DIFFERENCE, poly2.Polygon)
	}
	if typeval == "UNION" {
		poly1.Polygon.Construct(polyclip.UNION, poly2.Polygon)
	}

	ch <- true
}

func IsReachable(poly1 Polygon, poly2 Polygon, typeval string) bool {
	ch := make(chan bool, 2)
	go check(poly1, poly2, typeval, ch)

	time.AfterFunc(time.Millisecond*300, func() { ch <- false })
	return <-ch
}

func Checkpt(bds m.Extrema, pt polyclip.Point, ok bool) bool {
	if ok == true {
		bds.N = bds.N + .00001
		bds.E = bds.E + .00001
		bds.S = bds.S - .00001
		bds.W = bds.W - .00001

		if ((bds.N >= pt.Y) && (bds.S <= pt.Y)) && ((bds.E >= pt.X) && (bds.W <= pt.X)) {
			return false
		} else {
			return true
		}
	} else {

		if ((bds.N >= pt.Y) && (bds.S <= pt.Y)) && ((bds.E >= pt.X) && (bds.W <= pt.X)) {
			return false
		} else {
			return true
		}
	}
}

// taking the found polygons and returning a list of each intersected polygon
func Make_const_polygons(finds []Polygon, first Polygon, boolval bool) Output_Struct {
	newlist := []Polygon{}
	//result := polyclip.Polygon{}
	total := polyclip.Polygon{}

	// iterating through each found area
	// iterating through each found area
	for _, i := range finds {

		//if IsReachable(first, i, "INTERSECTION") == true {
		result := first.Polygon.Construct(polyclip.INTERSECTION, i.Polygon)
		//}
		for _, val := range result {
			total.Add(val)
		}

		// adding the the result to newlist if possible
		if len(result) != 0 {
			amap := map[string]string{}
			amap[first.Layer] = first.Area
			amap[i.Layer] = i.Area
			//fmt.Print(amap, "\n")
			i.Polygon = result
			i.Layers = amap
			newlist = append(newlist, i)
		} else {
			//	fmt.Print("here\n", first.Polystring, "\n", i.Polystring, "\n")
			//fmt.Print("here\n")
		}

	}

	//fmt.Print(newlist, "\n")
	if boolval == true {
		return Output_Struct{Polylist: newlist}
	} else if boolval == false {
		return Output_Struct{Polystring: Make_Polygon_String(newlist)}
	}
	return Output_Struct{Polystring: Make_Polygon_String(newlist)}

}

// makes a polygonial set in string in slice format
func Make_layer_polygon(first Polygon, layer2 []Polygon, boolval bool) Output_Struct {
	bds := first.Bounds
	finds := []Polygon{}
	//fmt.Print("N:", bds.N, " S:", bds.S, "\n")
	//bds.N = bds.N + .0001
	//bds.E = bds.E + .0001
	//bds.S = bds.S - .0001
	//bds.W = bds.W - .0001
	bds = m.Extrema{W: bds.W - 0.000001, S: bds.S - 0.000001, E: bds.E + 0.000001, N: bds.N + 0.000001}

	for _, polygon := range layer2 {
		testbds := polygon.Bounds
		//fmt.Print(bds.N, bds.S)

		c1 := []float64{testbds.E, testbds.N}
		c2 := []float64{testbds.W, testbds.N}
		c3 := []float64{testbds.E, testbds.S}
		c4 := []float64{testbds.W, testbds.S}
		valbool := false

		if (Checkpt(bds, polyclip.Point{c1[0], c1[1]}, false) == false) && (Checkpt(bds, polyclip.Point{c2[0], c2[1]}, false) == false) && (Checkpt(bds, polyclip.Point{c3[0], c3[1]}, false) == false) && (Checkpt(bds, polyclip.Point{c4[0], c4[1]}, false) == false) {
			valbool = true
		}

		if ((((bds.N > testbds.N) && (bds.S < testbds.N)) || ((bds.N > testbds.S) && (bds.S < testbds.S))) && (((bds.E > testbds.E) && (bds.W < testbds.E)) || ((bds.E > testbds.W) && (bds.W < testbds.W)))) || (valbool == true) {
			finds = append(finds, polygon)
		}

	}

	return Make_const_polygons(finds, first, boolval)
}

// this function will handle a polygon if it has
// more than 5 contours
// it will be used as the iteration value against the first polygon
func Lint_Polygon(poly Polygon, first Polygon) Polygon {
	bb := first.Polygon.BoundingBox()
	newpoly := polyclip.Polygon{}
	for _, cont := range poly.Polygon {
		if bb.Overlaps(cont.BoundingBox()) == true {
			newpoly.Add(cont)
		}
	}
	poly.Polygon = newpoly
	return poly
}
