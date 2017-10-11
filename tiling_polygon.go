package layersplit

// This part of the package consists of code to trim or envelope the polygons
// It also contains the subsequent code for creating polygon children.

import (
	//"fmt"
	m "github.com/murphy214/mercantile"
	pc "github.com/murphy214/polyclip"
	"github.com/paulmach/go.geojson"
)


// function for getting the extrema of an alignment
// it also converts the points from [][][]float64 > pc.Polygon
// in other words the clipping data structure
func get_extrema_coords(coords [][][]float64) (m.Extrema, pc.Polygon) {
	north := -1000.
	south := 1000.
	east := -1000.
	west := 1000.
	lat := 0.
	long := 0.
	polygon := pc.Polygon{}

	// iterating through each outer ring
	for _, coord := range coords {
		cont := pc.Contour{}
		// iterating through each point in a ring
		for _, i := range coord {
			lat = i[1]
			long = i[0]

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
			cont.Add(pc.Point{long, lat})

		}
		polygon.Add(cont)
	}

	return m.Extrema{S: south, W: west, N: north, E: east}, polygon

}


// gets the size of a tileid
func get_size(tile m.TileID) pc.Point {
	bds := m.Bounds(tile)
	return pc.Point{bds.E - bds.W, bds.N - bds.S}
}

// raw 1d linspace like found in numpy
func linspace(val1 float64, val2 float64, number int) []float64 {
	delta := (val2 - val1) / float64(number)
	currentval := val1
	newlist := []float64{val1}
	for currentval < val2 {
		currentval += delta
		newlist = append(newlist, currentval)
	}

	return newlist
}

// gets the middle of a tileid
func get_middle(tile m.TileID) pc.Point {
	bds := m.Bounds(tile)
	return pc.Point{(bds.E + bds.W) / 2.0, (bds.N + bds.S) / 2.0}
}

func grid_bounds(c2pt pc.Point, c4pt pc.Point, size pc.Point) m.Extrema {
	return m.Extrema{W: c2pt.X - size.X/2.0, N: c2pt.Y + size.Y/2.0, E: c4pt.X + size.X/2.0, S: c4pt.Y - size.Y/2.0}
}


// output structure to ensure everything stays in a key value stroe
type Output struct {
	Total [][][][]float64
	ID    m.TileID
}


// given a polygon to be tiled envelopes the polygon in corresponding boxes
func Env_Polygon(polygon *geojson.Feature, size int) map[m.TileID][]*geojson.Feature {
	// getting bds
	bds, poly := get_extrema_coords(polygon.Geometry.Polygon)
	properties := polygon.Properties
	id := polygon.ID


	// dummy values you know
	intval := 0
	tilemap := map[m.TileID][]int{}

	// getting all four corners
	c1 := pc.Point{bds.E, bds.N}
	c2 := pc.Point{bds.W, bds.N}
	c3 := pc.Point{bds.W, bds.S}
	c4 := pc.Point{bds.E, bds.S}

	// getting all the tile corners
	c1t := m.Tile(c1.X, c1.Y, size)
	c2t := m.Tile(c2.X, c2.Y, size)
	c3t := m.Tile(c3.X, c3.Y, size)
	c4t := m.Tile(c4.X, c4.Y, size)

	//tilemap := map[m.TileID][]int32{}
	tilemap[c1t] = append(tilemap[c1t], intval)
	tilemap[c2t] = append(tilemap[c2t], intval)
	tilemap[c3t] = append(tilemap[c3t], intval)
	tilemap[c4t] = append(tilemap[c4t], intval)
	sizetile := get_size(c1t)

	//c1pt := get_middle(c1t)
	c2pt := get_middle(c2t)
	//c3pt := get_middle(c3t)
	c4pt := get_middle(c4t)

	gridbds := grid_bounds(c2pt, c4pt, sizetile)
	//fmt.Print(gridbds, sizetile, "\n")
	sizepoly := pc.Point{bds.E - bds.W, bds.N - bds.S}
	xs := []float64{}
	if c2pt.X == c4pt.X {
		xs = []float64{c2pt.X}
	} else {
		xs = []float64{c2pt.X, c4pt.X}

	}
	ys := []float64{}
	if c2pt.Y == c4pt.Y {
		ys = []float64{c2pt.Y}
	} else {
		ys = []float64{c2pt.Y, c4pt.Y}

	}
	if sizetile.X < sizepoly.X {
		number := int((gridbds.E - gridbds.W) / sizetile.X)
		xs = linspace(gridbds.W, gridbds.E, number+1)
	}
	if sizetile.Y < sizepoly.Y {
		number := int((gridbds.N - gridbds.S) / sizetile.Y)
		ys = linspace(gridbds.S, gridbds.N, number+1)
	}

	//totallist := []string{}

	for _, xval := range xs {
		// iterating through each y
		for _, yval := range ys {
			tilemap[m.Tile(xval, yval, size)] = append(tilemap[m.Tile(xval, yval, size)], intval)
		}
	}
	c := make(chan Output)
	for k := range tilemap {
		go func(poly pc.Polygon, k m.TileID, c chan Output) {
			polys := Lint_Polygons(poly.Construct(pc.INTERSECTION, Make_Tile_Poly(k)))
			total := [][][][]float64{}
			for _, p := range polys {
				if len(p) > 0 { 
					total = append(total, Convert_Float(p))
				}

			}
			c <- Output{Total: total, ID: k}
		}(poly, k, c)
	}
	totalmap := map[m.TileID][]*geojson.Feature{}
	for range tilemap {
		output := <-c
		if len(output.Total) > 0 {
			for _, val := range output.Total {	
				newgeom := geojson.Geometry{Type: "Polygon"}
				newgeom.Polygon = val
				newfeat := geojson.Feature{Geometry: &newgeom, Properties: properties,ID:id}
				totalmap[output.ID] = append(totalmap[output.ID], &newfeat)

				//totalmap[output.ID] = append(totalmap[output.ID],  &geojson.Feature{Geometry: &geojson.Geometry{Type: "Polygon",Polygon:val}, Properties: polygon.Properties})
			
			}
		}
	}

	return totalmap

}

// makes the tile polygon
func Make_Tile_Poly(tile m.TileID) pc.Polygon {
	bds := m.Bounds(tile)
	return pc.Polygon{{pc.Point{bds.E, bds.N}, pc.Point{bds.W, bds.N}, pc.Point{bds.W, bds.S}, pc.Point{bds.E, bds.S}}}
}

// area of bds (of a square)
func AreaBds(ext m.Extrema) float64 {
	return (ext.N - ext.S) * (ext.E - ext.W)
}