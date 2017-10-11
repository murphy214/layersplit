package layersplit 

import (
	"math"
	"github.com/paulmach/go.geojson"
	m "github.com/murphy214/mercantile"
	"fmt"
	"io/ioutil"
	pc "github.com/murphy214/polyclip"
	g "github.com/murphy214/geobuf"

)



func Survey_Area_Ring(coords [][]float64) float64 {
	firstpt := coords[0]
	oldi := coords[0]
	totalarea := 0.0
	// oldi = x1
	for _,i := range coords[1:] {
		//
		totalarea += ((oldi[0] * i[1]) - (i[0] * oldi[1]))
		oldi = i
	}
	i := firstpt
	totalarea += ((oldi[0] * i[1]) - (i[0] * oldi[1]))

	totalarea = math.Abs(totalarea / 2.0)

	return totalarea
}

// returns the complete the survey area account for hoels in polygons
func Survey_Area(coords [][][]float64) float64 {
	newlist := []float64{}
	for _,i := range coords {
		newlist = append(newlist,Survey_Area_Ring(i))
	}

	totalarea := newlist[0]
	if len(newlist) > 1 {
		for _,i := range newlist[1:] {
			totalarea = totalarea - i
		}
	}
	return totalarea

}

// getting total area
func Get_Area_Total(feats []*geojson.Feature) float64 {
	c := make(chan float64)
	for _,feat := range feats {
		go func(feat *geojson.Feature,c chan float64) {
			c <- Survey_Area(feat.Geometry.Polygon)
		}(feat,c)
	}

	totalarea := 0.0
	for range feats {
		totalarea += <-c
	}

	return totalarea
}

// round bitch
func Round(val float64, roundOn float64, places int ) (newVal float64) {
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

type Feature_Map struct {
	Layer1Map map[m.TileID]map[interface{}]string
	Layer2Map map[m.TileID]map[interface{}]string
	SignatureMap map[m.TileID]string
	Size int
}	

func Get_Parent_Zoom(k m.TileID,desired_zoom int) m.TileID {
	for k.Z != uint64(desired_zoom) {
		k = Get_Parent(k)
	}
	return k
}

// gets the features from a given id
func Get_IDs(layer []*geojson.Feature,idmap map[interface{}]string) []*geojson.Feature {
	newlist := []*geojson.Feature{}
	for _,feat := range layer {
		_,ok := idmap[feat.ID]
		//fmt.Println(feat)
		if ok == true {
			newlist = append(newlist,feat)
		}

	}
	return newlist 
}
// creates a pc polygon
func Make_Polygon_Round(coords [][][]float64) pc.Polygon {
	thing2 := pc.Contour{}
	things := pc.Polygon{}
	for _, coord := range coords {
		thing2 = pc.Contour{}

		for _, i := range coord {
			if len(i) >= 2 {
				// moving sign in 10 ** -7 pla

				thing2.Add(pc.Point{X: Round(i[0],.5,6), Y: Round(i[1],.5,6)})
			}
		}
		if len(thing2) >= 2 {
			things.Add(thing2)
		}
	}

	return things
}


func To_Polyclip(feats []*geojson.Feature) []pc.Polygon {
	newlist := make([]pc.Polygon,len(feats))
	for ii,i := range feats {
		newlist[ii] = Make_Polygon(i.Geometry.Polygon)
	}
	return newlist
}

type Combine_Struct struct {
	Polygon pc.Polygon
	TileID m.TileID
}


// creates a new tileid polygon map for a given feature
func Combine_Upwards(newmap map[m.TileID][]pc.Polygon) map[m.TileID][]pc.Polygon {
	c := make(chan Combine_Struct)
	for k,v := range newmap {
		go func(k m.TileID,v []pc.Polygon,c chan Combine_Struct) {
			c <- Combine_Struct{Polygon:Make_Big(v),TileID:Get_Parent(k)}
		}(k,v,c)
	}

	newmap2 := map[m.TileID][]pc.Polygon{}
	for range newmap {
		out := <- c
		newmap2[out.TileID] = append(newmap2[out.TileID],out.Polygon)
	}

	return newmap2
}


// creates a feature map for a specific feature 
func Make_Feature_Map(feat_map map[m.TileID][]*geojson.Feature) []*geojson.Feature {
	var feature *geojson.Feature
	newmap := map[m.TileID][]pc.Polygon{}
	for k,v := range feat_map {
		newmap[k] = To_Polyclip(v)
		feature = v[0]
	}

	for len(newmap) > 1 {
		newmap = Combine_Upwards(newmap)	
	}

	var polygon pc.Polygon
	for _,v := range newmap {
		polygon = Make_Total_Big(v)
	}

	polygons := Lint_Polygons(polygon)
	feats := []*geojson.Feature{}
	for _,polygon := range polygons {
		if len(polygon) > 0 {
			coords := Convert_Float(polygon)
			if Survey_Area(coords) > 1e-12 {
				feats = append(feats,&geojson.Feature{Properties:feature.Properties,Geometry:&geojson.Geometry{Polygon:coords,Type:"Polygon"},ID:feature.ID})
			}
		}
	}
	return feats

}

func Append_Prefix_Layer(feats []*geojson.Feature,keys []string) []*geojson.Feature {
	for ii,idum := range feats {
		i := DeepCopy(idum)
		for _,k := range keys {
			i.Properties[k] = "NONE"
		}
		feats[ii] = i
	}
	return feats 
}


// creating the differences for each laeyr
func Make_Differences(layer1 []*geojson.Feature,layer2 []*geojson.Feature) ([]*geojson.Feature,[]*geojson.Feature) {
	// getting keys and keys2
	var keys1,keys2 []string
	for k := range layer1[0].Properties {
		keys1 = append(keys1,k)
	}
	for k := range layer2[0].Properties {
		keys2 = append(keys2,k)
	}

	// getting tilemap and tilemap2 	
	tilemap1 := Map_Layer(layer1,10)
	tilemap2 := Map_Layer(layer2,10)


	// creating one combined large tilemap
	tilemap := map[m.TileID]string{}
	for k := range tilemap1 {
		tilemap[k] = ""
	}
	for k := range tilemap2 {
		tilemap[k] = ""
	}

	var sema = make(chan struct{}, 1000)
	// DEBUG
	// DEBUG BOOL HERE!
	debug := false
	c := make(chan Dif_Output)
	// iterating through each value in a tilemap 
	// and determining the semantic relationship
	// based on that semantic relationship gets the layer differences
	// for each tile thats been split into tilemaps
	for k := range tilemap {

		go func(k m.TileID,c chan Dif_Output) {
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			// getting area square 
			areasquare := AreaBds(m.Bounds(k))
			var layer1area,layer2area float64

			// getting area layer1 
			vals1,ok := tilemap1[k]
			if ok == true {
				layer1area = Get_Area_Total(vals1)
			} else {
				layer1area = 0.0
			}

			// getting area layer1 
			vals2,ok := tilemap2[k]
			if ok == true {
				layer2area = Get_Area_Total(vals2)
			} else {
				layer2area = 0.0
			}
			var fullbool1,fullbool2 bool


			if Round(layer1area,.5,6) == Round(areasquare,.5,6) {
				fullbool1 = true
			} else {
				fullbool1 = false
			}

			if Round(layer2area,.5,6) == Round(areasquare,.5,6) {
				fullbool2 = true
			} else {
				fullbool2 = false
			}

			layer1dif := false
			layer2dif := false
			//var signature string
			if fullbool1 == true && layer2area > 0 && fullbool2 == false {
				layer1dif = true
				//signature = "layer2"
			} else if fullbool2 == true && layer1area > 0 && fullbool1 == false {
				layer2dif = true
				//signature = "layer1"
			} else if fullbool1 == false && fullbool2 == false && layer1area > 0 && layer2area > 0 {
				layer1dif = true
				layer2dif = true
				//signature = "both"
			} else if layer1area > 0 && layer2area == 0 {
				layer1dif = true
				//signature = "layer1"
			} else if layer2area > 0 && layer1area == 0 {
				layer2dif = true
				//signature = "layer2"
			}




			// appending shit
			if layer1dif == true || layer2dif == true {

				if debug == true {
					ioutil.WriteFile("a.buf",g.Make_FeatureCollection(vals1).Bytevals,0666)
					ioutil.WriteFile("b.buf",g.Make_FeatureCollection(vals2).Bytevals,0666)
				}
				if len(vals1) > 0 && len(vals2) > 0 {
					eh :=  Make_Tile_Differences(vals1,vals2,keys1,keys2)
					eh.TileID = k
					c <- eh
				} else if len(vals1) > 0 && len(vals2) == 0 {
					c <- Dif_Output{Layer1:Append_Prefix_Layer(vals1,keys2),TileID:k}
				} else if len(vals2) > 0 && len(vals1) == 0 {
					c <- Dif_Output{Layer2:Append_Prefix_Layer(vals2,keys1),TileID:k}
				} else {
				}

			} else {
				c <- Dif_Output{}
			}


		//fmt.Println(layer1dif,layer2dif)

		}(k,c)


		//fmt.Printf("Square: %f, Layer1: %f, Layer2: %f\n",areasquare,layer1area,layer2area)
	}


	// now collecting each from the output channel tilemap
	// it is collected and put into a map based on its id and feature context
	featmap1 := map[interface{}]map[m.TileID][]*geojson.Feature{}
	featmap2 := map[interface{}]map[m.TileID][]*geojson.Feature{}
	feats1 := []*geojson.Feature{}
	feats2 := []*geojson.Feature{}
	count := 0
	for range tilemap {
		out := <-c
		var parent m.TileID
		if len(out.Layer1) > 0 || len(out.Layer2) > 0 {
			parent = Get_Parent(out.TileID)

		}


		for _,feat := range out.Layer1 {
			if len(featmap1[feat.ID]) == 0 {
				featmap1[feat.ID] = map[m.TileID][]*geojson.Feature{}
			}

			featmap1[feat.ID][parent] = append(featmap1[feat.ID][parent],feat)
		}
		for _,feat := range out.Layer2 {
			if len(featmap2[feat.ID]) == 0 {
				featmap2[feat.ID] = map[m.TileID][]*geojson.Feature{}
			}

			featmap2[feat.ID][parent] = append(featmap2[feat.ID][parent],feat)
		}

		fmt.Printf("\r[%d/%d] Semantic Relationships.           ",count,len(tilemap))
		count += 1
	}


	// now creating the channel for combining the layer differences 
	// back together for layer1 features 
	cc := make(chan []*geojson.Feature)
	for _,v := range featmap1 {
		go func(v map[m.TileID][]*geojson.Feature,cc chan []*geojson.Feature) {
			eh := Make_Feature_Map(v)
			if len(eh) == 0 {
				//fmt.Println(v)
			}
			cc <- eh
		}(v,cc)

	}
	count = 0
	for range featmap1 {

		feats1 = append(feats1,<-cc...)
		count += 1
		fmt.Printf("\r[%d/%d] Collecting Features 1.           ",count,len(featmap1))
	}

	// now creating the channel for combining the layer differences 
	// back together for layer1 features 
	cc = make(chan []*geojson.Feature)
	for _,v := range featmap2 {
		go func(v map[m.TileID][]*geojson.Feature,cc chan []*geojson.Feature) {
			eh := Make_Feature_Map(v)
			if len(eh) == 0 {
				//fmt.Println(v)
			}
			cc <- eh
		}(v,cc)

	}
	count = 0
	for range featmap2 {
		feats2 = append(feats2,<-cc...)
		count += 1
		fmt.Printf("\r[%d/%d] Collecting Features 2.           ",count,len(featmap2))
	}

	return feats1,feats2 
}
