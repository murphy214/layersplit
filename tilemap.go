package layersplit

import (
	"fmt"
	"github.com/paulmach/go.geojson"
	m "github.com/murphy214/mercantile"
	pc "github.com/murphy214/polyclip"
	"sync"
	"math"
)

var mutex = &sync.Mutex{}


// gettting the tilemap of a layer feats
func Map_Layer(feats []*geojson.Feature, zoom int) map[m.TileID][]*geojson.Feature {
	tilemap := Make_Tilemap(&geojson.FeatureCollection{Features:feats},zoom-2)
	tilemap = Make_Tilemap_Children(tilemap)
	tilemap = Make_Tilemap_Children(tilemap)
	return tilemap
}

// given a polygon to be tiled envelopes the polygon in corresponding boxes
// from a polygon and a tileid return the tiles relating to the polygon 1 level lower
func Children_Polygon(polygon *geojson.Feature, tileid m.TileID) map[m.TileID][]*geojson.Feature {
	// getting bds
	bd, poly := get_extrema_coords(polygon.Geometry.Polygon)
	pt := poly[0][0]

	temptileid := m.Tile(pt.X, pt.Y, int(tileid.Z+1))
	bdtemp := m.Bounds(temptileid)

	// checking to see if the polygon lies entirely within a smaller childd
	if (bd.N <= bdtemp.N) && (bd.S >= bdtemp.S) && (bd.E <= bdtemp.E) && (bd.W >= bdtemp.W) {
		totalmap := map[m.TileID][]*geojson.Feature{}
		totalmap[temptileid] = append(totalmap[temptileid], polygon)
		return totalmap
	}

	// checking to see if the polygon is encompassed within a square
	bdtileid := m.Bounds(tileid)
	if (math.Abs(AreaBds(bdtileid)-AreaBds(bd)) < math.Pow(.000001,2.0)) && len(poly) == 1 && len(poly[0]) == 4 {
		//fmt.Print("here\n")
		totalmap := map[m.TileID][]*geojson.Feature{}

		tiles := m.Children(tileid)
		for _, k := range tiles {
			//poly := Make_Tile_Poly(k)
			bds := m.Bounds(k)
			poly := [][][]float64{{{bds.E, bds.N}, {bds.W, bds.N}, {bds.W, bds.S}, {bds.E, bds.S}}}
			newgeom := geojson.Geometry{Type: "Polygon", Polygon: poly}

			totalmap[k] = append(totalmap[k], &geojson.Feature{Geometry: &newgeom, Properties: polygon.Properties,ID:polygon.ID})
		}

		return totalmap

	}

	//fmt.Print("\r", len(polygon.Geometry.Polygon[0]))

	c := make(chan Output)
	// creating the 4 possible children tiles
	// and sending into a go function
	tiles := m.Children(tileid)
	for _, k := range tiles {
		newpoly := poly
		go func(newpoly pc.Polygon, k m.TileID, c chan Output) {
			newpoly2 := newpoly.Construct(pc.INTERSECTION, Make_Tile_Poly(k))
			polys := Lint_Polygons(newpoly2)
			total := [][][][]float64{}
			for _, p := range polys {
				total = append(total, Convert_Float(p))

			}
			c <- Output{Total: total, ID: k}
		}(newpoly, k, c)
	}
	totalmap := map[m.TileID][]*geojson.Feature{}
	properties := polygon.Properties
	for range tiles {
		output := <-c
		if len(output.Total) > 0 {
			for _, coord := range output.Total {
				newgeom := geojson.Geometry{Type: "Polygon"}
				newgeom.Polygon = coord
				newfeat := geojson.Feature{Geometry: &newgeom, Properties: properties,ID:polygon.ID}
				totalmap[output.ID] = append(totalmap[output.ID], &newfeat)
			}
		}
	}

	return totalmap

}


// makes children and returns tilemap of a first intialized tilemap
func Make_Tilemap_Children(tilemap map[m.TileID][]*geojson.Feature) (map[m.TileID][]*geojson.Feature) {
	// iterating through each tileid
	ccc := make(chan map[m.TileID][]*geojson.Feature)
	newmap := map[m.TileID][]*geojson.Feature{}
	count2 := 0
	counter := 0
	sizetilemap := len(tilemap)
	buffer := 100000

	// iterating through each tielmap
	for k, v := range tilemap {
		go func(k m.TileID, v []*geojson.Feature, ccc chan map[m.TileID][]*geojson.Feature) {
			cc := make(chan map[m.TileID][]*geojson.Feature)
			for _, i := range v {
				go func(k m.TileID, i *geojson.Feature, cc chan map[m.TileID][]*geojson.Feature) {
					if i.Geometry.Type == "Polygon" {
						cc <- Children_Polygon(i, k)
					} else if i.Geometry.Type == "LineString" {
						//partmap := Env_Line(i, int(k.Z+1))
						//partmap = Lint_Children_Lines(partmap, k)
						//cc <- partmap
					} else if i.Geometry.Type == "Point" {
						//partmap := map[m.TileID][]*geojson.Feature{}
						//pt := i.Geometry.Point
						//tileid := m.Tile(pt[0], pt[1], int(k.Z+1))
						//partmap[tileid] = append(partmap[tileid], i)
						//cc <- partmap
					}
				}(k, i, cc)
			}

			// collecting all into child map
			childmap := map[m.TileID][]*geojson.Feature{}
			for range v {
				tempmap := <-cc
				for k, v := range tempmap {
					childmap[k] = append(childmap[k], v...)
				}
			}

			ccc <- childmap
		}(k, v, ccc)

		counter += 1
		// collecting shit
		if (counter == buffer) || (sizetilemap-1 == count2) {
			count := 0

			for count < counter {
				tempmap := <-ccc
				for k, v := range tempmap {
					newmap[k] = append(newmap[k], v...)
				}
				count += 1
			}
			counter = 0
			fmt.Printf("\r[%d / %d] Tiles Complete, Size: %d           ", count2, sizetilemap, int(k.Z)+1)

		}
		count2 += 1

	}


	// getting size of total number of features within the tilemap
	totalsize := 0
	for _,v := range newmap {
		totalsize += len(v)
	}

	return newmap
}

// makes a tilemap and returns
func Make_Tilemap(feats *geojson.FeatureCollection, size int) map[m.TileID][]*geojson.Feature {
	c := make(chan map[m.TileID][]*geojson.Feature)
	for _, i := range feats.Features {
		partmap := map[m.TileID][]*geojson.Feature{}

		go func(i *geojson.Feature, size int, c chan map[m.TileID][]*geojson.Feature) {
			//partmap := map[m.TileID][]*geojson.Feature{}

			if i.Geometry.Type == "Polygon" {
				partmap = Env_Polygon(i, size)
			} else if i.Geometry.Type == "LineString" {
				//partmap = Env_Line(i, size)
			} else if i.Geometry.Type == "Point" {
				//pt := i.Geometry.Point
				///tileid := m.Tile(pt[0], pt[1], size)
				//partmap[tileid] = append(partmap[tileid], i)
			}
			c <- partmap
		}(i, size, c)
	}

	// collecting channel shit
	totalmap := map[m.TileID][]*geojson.Feature{}
	sizetilemap := len(feats.Features)
	for i := range feats.Features {
		partmap := <-c
		for k, v := range partmap {
			totalmap[k] = append(totalmap[k], v...)
		}
		fmt.Printf("\r[%d / %d] Tiles Complete, Size: %d           ",i,sizetilemap,size)
	}

	return totalmap
}


// f
func Make_Combined(layer []*geojson.Feature) pc.Polygon {
	newlist := []pc.Polygon{}
	for _,i := range layer {
		newlist = append(newlist,Make_Polygon(i.Geometry.Polygon))
	}
	return Make_Big(newlist)
}

// makes the differences between the two tiles
func Make_Difference_Tile(layer []*geojson.Feature,big pc.Polygon,keys []string) []*geojson.Feature {
	feats := []*geojson.Feature{}

	for _,idum := range layer {
		i := DeepCopy(idum)

		ipoly := Make_Polygon(idum.Geometry.Polygon)
		result := ipoly.Construct(pc.DIFFERENCE,big)
		polygons := Lint_Polygons(result)
		//ii := new(geojson.Feature)
		//ii := &i
		for _,k := range keys {
			i.Properties[k] = "NONE"
		}

		for _,polygon := range polygons {
			if len(polygon) > 0 {
				feats = append(feats,&geojson.Feature{Properties:i.Properties,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"},ID:i.ID})
			}
		}
	}
	return feats
}


// differnent output for each layer differences
type Dif_Output struct {
	Layer1 []*geojson.Feature
	Layer2 []*geojson.Feature
	TileID m.TileID

}


// 
func Make_Tile_Differences(layer1 []*geojson.Feature,layer2 []*geojson.Feature,keys1 []string,keys2 []string) Dif_Output {

	big1 := Make_Combined(layer1)
	big2 := Make_Combined(layer2)

	///feat1 := &geojson.Feature{Geometry:&geojson.Geometry{Type:"Polygon",Polygon:Convert_Float(big1)},Properties:map[string]interface{}{"AREA":"1"}}
	//feat2 := &geojson.Feature{Geometry:&geojson.Geometry{Type:"Polygon",Polygon:Convert_Float(big2)},Properties:map[string]interface{}{"AREA":"2"}}
	//fc := &geojson.FeatureCollection{Features:[]*geojson.Feature{feat1,feat2}}
	//shit,_ := fc.MarshalJSON()
	//ioutil.WriteFile("gf/new.geojson",[]byte(shit),0666)


	totalfeats := Make_Difference_Tile(layer1,big2,keys2)

	totalfeats2 := Make_Difference_Tile(layer2,big1,keys1)

	return Dif_Output{Layer1:totalfeats,Layer2:totalfeats2}
}


func Get_Parent(k m.TileID) m.TileID {
	middle := get_middle(k)
	return m.Tile(middle.X,middle.Y,int(k.Z)-1)
}