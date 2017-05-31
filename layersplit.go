package layersplit

import (
	"./polyclip"
	"encoding/json"
	"fmt"
	//"math/rand"
	"io/ioutil"
	m "mercantile"
	p "newmod/poly_index"
	"os/exec"
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
}

type ResponseCoords2 struct {
	Coords [][][]float64 `json:"coords"`
}

// gets the slope of two points along a line
// if statement logic accounts for undefined corner case
func get_slope(pt1 polyclip.Point, pt2 polyclip.Point) float64 {
	if pt1.X == pt2.X {
		return 1000000.0
	}
	return (pt2.Y - pt1.Y) / (pt2.X - pt1.X)
}

// iteroplates the position of y based on x of the location between two points
// this function accepts m the slope to keep it from recalculating
// what could be several hundred/thousand times between two points
func interp(pt1 polyclip.Point, pt2 polyclip.Point, m float64, x float64) polyclip.Point {
	y := (x-pt1.X)*m + pt1.Y
	return polyclip.Point{x, y}
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
func make_layer(polys [][]string, layername string) []Polygon {
	layer := []Polygon{}

	c := make(chan []Polygon)
	for _, polyrow := range polys {
		polygonstrings := strings.Split(polyrow[1], "|")
		go func(polygonstrings []string, polyrow []string, c chan<- []Polygon) {
			templayer := []Polygon{}
			for _, polygonstring := range polygonstrings {

				polygon := get_coords_json2(polygonstring)
				polygonc := make_polygon(polygon)
				extrema := get_extrema_coords(polygon[0])
				templayer = append(templayer, Polygon{Layer: layername, Polygon: polygonc, Bounds: extrema, Polystring: polygonstring, Area: string(polyrow[0])})
			}
			c <- templayer
		}(polygonstrings, polyrow, c)
		//for _, polygonstring := range polygonstrings {
		//	go func(polygonstring string, polyrow []string, c chan<- Polygon) {
		//		polygon := get_coords_json2(polygonstring)
		//		polygonc := make_polygon(polygon)
		//		extrema := get_extrema_coords(polygon[0])

		//	}(polygonstring, polyrow, c)
		//	layer = append(layer, Polygon{Area: polyrow[0], Layer: layername, Polygon: polygonc, Bounds: extrema, Polystring: polygonstring})

	}
	for ii := 0; ii < len(polys); ii++ {
		select {
		case msg1 := <-c:
			layer = append(layer, msg1...)

		}
	}

	return layer
}

func reverse(poly polyclip.Contour) polyclip.Contour {
	if len(poly) <= 1 {
		return poly
	}
	current := len(poly) - 1
	newpoly := polyclip.Contour{}
	for current != -1 {
		newpoly.Add(poly[current])
		current = current - 1
	}
	return newpoly
}

func Correct_coords(coords polyclip.Contour) polyclip.Contour {
	needstobe := "positive"
	count := 0
	value := float64(0)
	oldpt := polyclip.Point{}
	var val string
	var firstpt polyclip.Point
	for _, pt := range coords {
		if count == 0 {
			count = 1
			firstpt = pt
		} else {
			//value += (4096.0 - float64(oldpt[1]) + (4096.0 - float64(pt[1]))) * (float64(pt[0]) - float64(oldpt[0]))
			value += (float64(oldpt.Y) + (float64(pt.Y))) * (float64(pt.X) - float64(oldpt.X))

			count += 1
		}
		oldpt = pt
	}
	pt := firstpt

	//value += (4096.0 - float64(oldpt[1]) + (4096.0 - float64(pt[1]))) * (float64(pt[0]) - float64(oldpt[0]))
	value += (float64(oldpt.Y) + (float64(pt.Y))) * (float64(pt.X) - float64(oldpt.X))

	fmt.Print(count, len(coords), "\n")
	if value <= 0 {
		val = "positive"
	} else {
		val = "negative"
	}

	if val == needstobe {
		return coords
	} else {
		return reverse(coords)
	}
}

func assert_winding_order(polygon polyclip.Polygon) polyclip.Polygon {
	newpolygon := polyclip.Polygon{}
	for _, cont := range polygon {
		newpolygon.Add(Correct_coords(cont))
	}
	return newpolygon
}

func Within(poly1 polyclip.Contour, poly2 polyclip.Contour) bool {

	boolval := true
	for _, pt := range poly1 {
		if poly2.Contains(pt) == false {
			boolval = false
			return boolval
		}
	}
	return boolval
}

func make_polygons(polygon polyclip.Polygon) []polyclip.Polygon {

	//e := polyclip.Polygon{}
	mymap := map[int]polyclip.Polygon{}
	for i, cont := range polygon {
		mymap[i] = polyclip.Polygon{cont}

	}

	for oldi, oldcont := range polygon {
		for i, cont := range polygon {
			if (Within(oldcont, cont) == true) && (oldi != i) {
				values := mymap[i]
				values.Add(oldcont)
				mymap[i] = values
				delete(mymap, oldi)
			}
		}

	}

	//fmt.Print(len(mymap), len(polygon), "\n")
	newlist := []polyclip.Polygon{}
	for _, v := range mymap {
		newlist = append(newlist, v)
	}

	//fmt.Print(len(e))
	//fmt.Print(len(nonp), len(maxp), "\n")
	return newlist
}

// finding the point that intersects with a given y
func opp_interp(pt1 polyclip.Point, pt2 polyclip.Point, y float64) polyclip.Point {
	m := get_slope(pt1, pt2)
	x := ((y - pt1.Y) / m) + pt1.X
	return polyclip.Point{x, y}
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

func checkpt(bds m.Extrema, pt polyclip.Point, ok bool) bool {
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

func raw_polygons(first Polygon, layer2 []Polygon) string {
	bds := first.Bounds
	finds := []Polygon{}
	//fmt.Print("N:", bds.N, " S:", bds.S, "\n")
	bds.N = bds.N + .0001
	bds.E = bds.E + .0001
	bds.S = bds.S - .0001
	bds.W = bds.W - .0001
	for _, polygon := range layer2 {
		testbds := polygon.Bounds
		//fmt.Print(bds.N, bds.S)

		c1 := []float64{testbds.E, testbds.N}
		c2 := []float64{testbds.W, testbds.N}
		c3 := []float64{testbds.E, testbds.S}
		c4 := []float64{testbds.W, testbds.S}
		valbool := false

		if (checkpt(bds, polyclip.Point{c1[0], c1[1]}, false) == false) && (checkpt(bds, polyclip.Point{c2[0], c2[1]}, false) == false) && (checkpt(bds, polyclip.Point{c3[0], c3[1]}, false) == false) && (checkpt(bds, polyclip.Point{c4[0], c4[1]}, false) == false) {
			valbool = true
		}

		if ((((bds.N > testbds.N) && (bds.S < testbds.N)) || ((bds.N > testbds.S) && (bds.S < testbds.S))) && (((bds.E > testbds.E) && (bds.W < testbds.E)) || ((bds.E > testbds.W) && (bds.W < testbds.W)))) || (valbool == true) {
			// && (((bds.E > testbds.E) && (bds.W < testbds.E)) || ((bds.E > testbds.W) && (bds.W < testbds.W))) {
			//fmt.Print("S:", bds.S, " N:", bds.N, "test gap", testbds.S, testbds.N, polygon.Polystring[:100], "\n")
			finds = append(finds, polygon)
		}
		//if ((bds.N < testbds.N) && (bds.S > testbds.N)) || ((bds.N < testbds.S) && (bds.S > testbds.S)) {
		// && (((bds.E > testbds.E) && (bds.W < testbds.E)) || ((bds.E > testbds.W) && (bds.W < testbds.W))) {

	}
	newlist := []string{fmt.Sprintf(`%s,"%s"`, first.Area, first.Polystring)}
	for _, newpoly := range finds {
		newlist = append(newlist, fmt.Sprintf(`%s,"%s"`, newpoly.Area, newpoly.Polystring))
	}

	return strings.Join(newlist, "\n")
}

func raw_polygon(first polyclip.Polygon, area string) string {
	if len(first) == 0 {
		return ""
	}
	newlist := []string{}
	for _, newpoly := range first {
		newlist2 := []string{}
		for _, row := range newpoly {
			newlist2 = append(newlist2, fmt.Sprintf("[%f,%f]", row.X, row.Y))
		}
		newlist = append(newlist, fmt.Sprintf("[%s]", strings.Join(newlist2, ",")))
	}
	return fmt.Sprintf(`%s,"[%s]"`, area, strings.Join(newlist, ","))
}

func write(strval string) {
	_ = ioutil.WriteFile("d.csv", []byte(strval), 0644)
}

func wepython(code []string) {
	start := "import pandas as pd\nimport numpy as np\nimport mapkit as mk\n"

	total := start + strings.Join(code, "\n")

	_ = ioutil.WriteFile("a.py", []byte(total), 0644)
	cmd := exec.Command("killall", "Safari")

	cmd = exec.Command("freeport", "8000")

	cmd = exec.Command("python", "a.py")
	stdoutStderr, _ := cmd.CombinedOutput()

	fmt.Printf("%s\n", stdoutStderr)

}

func make_const_polygons(finds []Polygon, first Polygon) []Polygon {
	newlist := []Polygon{}
	result := polyclip.Polygon{}
	total := polyclip.Polygon{}

	newmap := map[string]string{}
	newmap[first.Layer] = first.Area
	if len(finds) > 0 {
		newmap[finds[0].Layer] = "NONE"
	}
	for _, i := range finds {
		amap := map[string]string{}
		amap[first.Layer] = first.Area

		amap[i.Layer] = i.Area

		if IsReachable(first, i, "DIFFERENCE") == true {
			result = first.Polygon.Construct(polyclip.INTERSECTION, i.Polygon)
			for _, val := range result {
				total.Add(val)
			}

			if len(result) == 0 {
				if IsReachable(first, i, "DIFFERENCE") == true {
					result = i.Polygon.Construct(polyclip.INTERSECTION, first.Polygon)
					for _, val := range result {
						total.Add(val)
					}
				}
			}

		}
		if len(result) != 0 {
			bb := result.BoundingBox()
			bds := m.Extrema{E: bb.Max.X, W: bb.Min.X, N: bb.Max.Y, S: bb.Min.Y}

			newlist = append(newlist, Polygon{Polygon: result, Layers: amap, Bounds: bds})
		}
	}
	addtotal := true
	if addtotal == true { //&& (len(total) != 0) {
		newtotalpolygon := polyclip.Polygon{first.Polygon[0]}
		for _, cc := range total {
			newtotalpolygon.Add(cc)
		}

		bb := newtotalpolygon.BoundingBox()
		bds := m.Extrema{E: bb.Max.X, W: bb.Min.X, N: bb.Max.Y, S: bb.Min.Y}

		newlist = append(newlist, Polygon{Polygon: newtotalpolygon, Layers: newmap, Bounds: bds})

	} else {
		newtotalpolygon := polyclip.Polygon{}
		newlist = append(newlist, Polygon{Polygon: newtotalpolygon, Layers: newmap})

	}

	return newlist

}

func polystring(polys []Polygon, layername string) string {
	newlist := []string{}
	c := make(chan string)

	keys := []string{}
	for k := range polys[0].Layers {
		keys = append(keys, k)
	}
	for _, row := range polys {
		go func(row Polygon, layername string, c chan string) {
			if layername == "Layers" {
				val := row.Layers

				newv := []string{}
				for _, k := range keys {
					newv = append(newv, val[k])
				}
				v := strings.Join(newv, ",")
				c <- raw_polygon(row.Polygon, v)

			} else {
				c <- raw_polygon(row.Polygon, row.Layers[layername])
			}
		}(row, layername, c)
	}
	count := 0
	total := 0
	for range polys {
		select {
		case msg1 := <-c:
			if count == 100 {
				count = 0
				total += 100
				fmt.Print(total, "\n")
			}
			newlist = append(newlist, msg1)
			count += 1
		}
	}
	return strings.Join(newlist, "\n")
}

func make_layer_polygon(first Polygon, layer2 []Polygon) []Polygon {
	bds := first.Bounds
	finds := []Polygon{}
	//fmt.Print("N:", bds.N, " S:", bds.S, "\n")
	bds.N = bds.N + .0001
	bds.E = bds.E + .0001
	bds.S = bds.S - .0001
	bds.W = bds.W - .0001
	for _, polygon := range layer2 {
		testbds := polygon.Bounds
		//fmt.Print(bds.N, bds.S)

		c1 := []float64{testbds.E, testbds.N}
		c2 := []float64{testbds.W, testbds.N}
		c3 := []float64{testbds.E, testbds.S}
		c4 := []float64{testbds.W, testbds.S}
		valbool := false

		if (checkpt(bds, polyclip.Point{c1[0], c1[1]}, false) == false) && (checkpt(bds, polyclip.Point{c2[0], c2[1]}, false) == false) && (checkpt(bds, polyclip.Point{c3[0], c3[1]}, false) == false) && (checkpt(bds, polyclip.Point{c4[0], c4[1]}, false) == false) {
			valbool = true
		}

		if ((((bds.N > testbds.N) && (bds.S < testbds.N)) || ((bds.N > testbds.S) && (bds.S < testbds.S))) && (((bds.E > testbds.E) && (bds.W < testbds.E)) || ((bds.E > testbds.W) && (bds.W < testbds.W)))) || (valbool == true) {
			finds = append(finds, polygon)
		}

	}
	return make_const_polygons(finds, first)
}

// Given two slices containg two different geographic layers
// returns 1 slice representing the two layers in which there are split into
// smaller polygons for creating hiearchies e.g. slicing zip codes about counties
func Combine_Layers(layer1 []Polygon, layer2 []Polygon) []Polygon {
	c := make(chan []Polygon)
	newlist := []Polygon{}
	for _, row := range layer1 {
		go func(row Polygon, layer2 []Polygon, c chan<- []Polygon) {
			a := make_layer_polygon(row, layer2)
			//fmt.Print(a, "\n")
			c <- a
		}(row, layer2, c)
	}
	count := 0
	total := 0
	for range layer1 {
		select {
		case msg1 := <-c:
			if count == 100 {
				count = 0
				total += 100
				fmt.Print(total, "\n")
			}
			newlist = append(newlist, msg1...)
			count += 1

		}
	}
	return newlist
}
