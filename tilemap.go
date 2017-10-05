package layersplit

import (
	"fmt"
	"github.com/paulmach/go.geojson"
	m "github.com/murphy214/mercantile"
	"github.com/murphy214/gotile/gotile"
	pc "github.com/murphy214/polyclip"
	//"io/ioutil"
	"sync"
)

var mutex = &sync.Mutex{}


// gettting the tilemap of a layer feats
func Map_Layer(feats []*geojson.Feature, zoom int) map[m.TileID][]*geojson.Feature {
	tilemap := Make_Tilemap(&geojson.FeatureCollection{Features:feats},zoom-2)
	tilemap = Make_Tilemap_Children(tilemap)
	tilemap = Make_Tilemap_Children(tilemap)

	return tilemap
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
						cc <- tile_surge.Children_Polygon(i, k)
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
			fmt.Printf("\r[%d / %d] Tiles Complete, Size: %d       ", count2, sizetilemap, int(k.Z)+1)

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
	for range feats.Features {
		partmap := <-c
		for k, v := range partmap {
			totalmap[k] = append(totalmap[k], v...)
		}
	}

	// getting size of total number of features within the tilemap
	totalsize := 0
	for _,v := range totalmap {
		totalsize += len(v)
	}


	//filemap := TileMapIO(totalmap)
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
		ipoly := Make_Polygon(idum.Geometry.Polygon)
		result := ipoly.Construct(pc.DIFFERENCE,big)
		polygons := Lint_Polygons(result)
		//ii := new(geojson.Feature)
		//ii := &i
		i := DeepCopy(idum)
		for _,k := range keys {
			i.Properties[k] = "NONE"
		}

		for _,polygon := range polygons {
			feats = append(feats,&geojson.Feature{Properties:i.Properties,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"}})
		}
	}
	return feats
}

// 
func Make_Tile_Differences(layer1 []*geojson.Feature,layer2 []*geojson.Feature,keys1 []string,keys2 []string) []*geojson.Feature {

	big1 := Make_Combined(layer1)
	big2 := Make_Combined(layer2)

	///feat1 := &geojson.Feature{Geometry:&geojson.Geometry{Type:"Polygon",Polygon:Convert_Float(big1)},Properties:map[string]interface{}{"AREA":"1"}}
	//feat2 := &geojson.Feature{Geometry:&geojson.Geometry{Type:"Polygon",Polygon:Convert_Float(big2)},Properties:map[string]interface{}{"AREA":"2"}}
	//fc := &geojson.FeatureCollection{Features:[]*geojson.Feature{feat1,feat2}}
	//shit,_ := fc.MarshalJSON()
	//ioutil.WriteFile("gf/new.geojson",[]byte(shit),0666)


	totalfeats2 := Make_Difference_Tile(layer1,big2,keys2)

	totalfeats := Make_Difference_Tile(layer2,big1,keys1)

	return append(totalfeats,totalfeats2...)
}

// combines the differences of the two layers
func Make_Differences(tilemap1 map[m.TileID][]*geojson.Feature,tilemap2 map[m.TileID][]*geojson.Feature,keys1 []string,keys2 []string) []*geojson.Feature {
	tilemap := map[m.TileID]string{}
	for k := range tilemap1 {
		tilemap[k] = ""
	}
	for k := range tilemap2 {
		tilemap[k] = ""
	}

	fmt.Print("\n")
	feats := []*geojson.Feature{}
	c := make(chan []*geojson.Feature)
	var sema = make(chan struct{}, 1000)
	unreached := 0
	// iterating through all the keys
	for k := range tilemap {
		go func(k m.TileID,c chan []*geojson.Feature) {
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			val1,ok1 := tilemap1[k]
			val2,ok2 := tilemap2[k]

			if ok1 == true && ok2 == true {
				c <- Make_Tile_Differences(val1,val2,keys1,keys2)
				
			} else if ok1 == false && ok2 == false {
				c <- []*geojson.Feature{}
			} else if ok1 == false {
				c <- val2
			} else if ok2 == false {
				c <- val1
			} else {
				c <- []*geojson.Feature{}
			}

		}(k,c)
	}
	count := 0
	for range tilemap {
		feats = append(feats,<-c...)
		fmt.Printf("\r Creating Non-Intersecting Features [%d/%d]",count,len(tilemap))

		count += 1
	}
	fmt.Print("\n",unreached,"\n")

	fmt.Print("\n")


	return feats
}




