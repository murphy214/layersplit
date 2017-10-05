package layersplit

import (
	"github.com/paulmach/go.geojson"
	pc "github.com/murphy214/polyclip"
	m "github.com/murphy214/mercantile"
	"fmt"
	"time"
	"math/rand"
)

// structure for creaitng output
type Output_Feature struct {
	Polygon pc.Polygon
	Feature *geojson.Feature
	BB m.Extrema
}

type Layer struct {
	Features []*geojson.Feature
	BBs []m.Extrema
	Polygons []pc.Polygon
}

// structure for a map output
type Map_Output struct {
	Key Output_Feature
	Feats []Output_Feature
}



func DeepCopy(a *geojson.Feature) *geojson.Feature {
	mymap := map[string]interface{}{}
	ehmap := a.Properties
	for k,v := range ehmap {
		mymap[k] = v
	}



	geometry := &geojson.Geometry{}
	*geometry = *a.Geometry


	aa := &geojson.Feature{Properties:mymap,Geometry:geometry}

	return aa
	

}

// creates a pc polygon
func Make_Polygon(coords [][][]float64) pc.Polygon {
	thing2 := pc.Contour{}
	things := pc.Polygon{}
	for _, coord := range coords {
		thing2 = pc.Contour{}

		for _, i := range coord {
			if len(i) >= 2 {
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
				thing2.Add(pc.Point{X: i[0]+factor1, Y: i[1]+factor2})
			}
		}
		if len(thing2) >= 2 {
			things.Add(thing2)
		}
	}

	return things
}

// gets the size of a coord
func Get_Size(coords [][][] float64) int {
	total := 0
	for _,i := range coords {
		total += len(i)
	}
	return total
}
 

// function for splitting up large features into small oens
func Split_Large_Features(fc *geojson.FeatureCollection) *geojson.FeatureCollection {
	newlist := []*geojson.Feature{}
	newlist2 := []*geojson.Feature{}

	for _,i := range fc.Features {
		coords := i.Geometry.Polygon
		if Get_Size(coords) > 1000 {
			newlist = append(newlist,i)
		} else {
			newlist2 = append(newlist2,i)
		}
	}

	newmap := Map_Layer(newlist,8)

	newlist = []*geojson.Feature{}
	for _,i := range newmap {
		newlist = append(newlist,i...)
	}
	newlist = append(newlist,newlist2...)

	fc.Features = newlist
	return fc
}




// creates a structure that will be representitive of a layer
func Make_Structs(gjson *geojson.FeatureCollection,prefix string) Layer {
	gjson = Split_Multi(gjson)
	gjson = Split_Large_Features(gjson)
	c := make(chan Output_Feature)
	for _,i := range gjson.Features {
		go func(i *geojson.Feature,c chan Output_Feature) {
			if prefix != "NONE" {
				i.Properties = Add_Prefix(i.Properties,prefix)
			}
			feat := Output_Feature{Polygon:Make_Polygon(i.Geometry.Polygon),Feature:i}
			bb := feat.Polygon.BoundingBox()
			feat.BB = m.Extrema{W:bb.Min.X,E:bb.Max.X,S:bb.Min.Y,N:bb.Max.Y}
			c <- feat
		}(i,c)

	}
	bbs := []m.Extrema{}
	feats := []*geojson.Feature{}
	polygons := []pc.Polygon{}
	for range gjson.Features {
		val := <- c
		if len(val.Polygon) > 0 {
			if len(val.Polygon[0]) > 0 {
				bbs = append(bbs,val.BB)
				feats = append(feats,val.Feature)
				polygons = append(polygons,val.Polygon)
			}
		}
	}

	return Layer{Features:feats,BBs:bbs,Polygons:polygons}
}

// structure for finding overlapping values
func Overlapping_1D(box1min float64,box1max float64,box2min float64,box2max float64) bool {
	if box1max >= box2min && box2max >= box1min {
		return true
	} else {
		return false
	}
	return false
}


// returns a boolval for whether or not the bb intersects
func Intersect(bdsref m.Extrema,bds m.Extrema) bool {
	if Overlapping_1D(bdsref.W-.0000001,bdsref.E+.0000001,bds.W-.0000001,bds.E+.0000001) && Overlapping_1D(bdsref.S-.0000001,bdsref.N+.0000001,bds.S-.0000001,bds.N+.0000001) {
		return true
	} else {
		return false
	}

	return false
}

// creates bounding box featues
func Create_BB_Feats(feat *geojson.Feature,bb m.Extrema,layer2 Layer) ([]int) {
	feats := []int{}
	for ii,bb2 := range layer2.BBs {
		boolval := Intersect(bb,bb2)
		if boolval == true {
			feats = append(feats,ii)
		}
	}

	return feats
}

// adds a prefix to each key in a map
func Add_Prefix(map1 map[string]interface{},prefix string) map[string]interface{} {
	newmap := map[string]interface{}{}
	for k,v := range map1 {
		newmap[prefix + "_" + k] = v
	}
	return newmap
}

// Returns the newly created map
func Combine_Properties(map1 map[string]interface{},map2 map[string]interface{}) map[string]interface{} {
	newmap := map1
	for k,v := range map2 {
		newmap[k] = v
	}
	return newmap
}


func Copy_Feat(usera *geojson.Feature, userb *geojson.Feature) {

    *userb = *usera
    fmt.Println(&userb,&usera)


}

// 
func Filter_FC(fc *geojson.FeatureCollection,keys []string) *geojson.FeatureCollection {
	for ii,i := range fc.Features {
		mymap := map[string]interface{}{}
		for _,k := range keys {
			mymap[k] = i.Properties[k]
		}
		i.Properties = mymap
		fc.Features[ii] = i
	}
	return fc
}






// taking the found polygons and returning a list of each intersected polygon
func Make_const_polygons(layer Layer) []*geojson.Feature {
	first := layer.Polygons[0]
	first_feature := layer.Features[0]
	layer.Features = layer.Features[1:]
	layer.BBs = layer.BBs[1:]
	layer.Polygons = layer.Polygons[1:]

	c := make(chan []*geojson.Feature)
	for i, val := range layer.Features {
		go func(i int,val *geojson.Feature,c chan []*geojson.Feature) {
			tempfeats := []*geojson.Feature{}
			newpoly := layer.Polygons[i]
			feat := DeepCopy(val)
			result := first.Construct(pc.INTERSECTION, newpoly)
			// adding the the result to newlist if possible
			if len(result) != 0 {

				for k,v := range first_feature.Properties {
					feat.Properties[k] = v
				}
	 			polygons := Lint_Polygons(result)
				for _,polygon := range polygons {

					mine := &geojson.Feature{Properties:feat.Properties,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"}}				
					tempfeats = append(tempfeats,mine)

				}
			} 
			c <- tempfeats
		}(i,val,c)
	}

	feats := []*geojson.Feature{}
	for range layer.Features {
		feats = append(feats,<-c...)
	}



	return feats
}


func check(i Layer, ch chan<- bool) {
	Make_const_polygons(i)
	ch <- true
}

func IsReachable(i Layer) bool {
	ch := make(chan bool, 2)
	go check(i, ch)

	time.AfterFunc(time.Second*25, func() { ch <- false })
	return <-ch
}

func check2(layer1 []*geojson.Feature,layer2 []*geojson.Feature,keys1 []string,keys2 []string,ch chan<- bool) {
	Make_Tile_Differences(layer1,layer2,keys1,keys2)
	ch <- true
}




func IsReachable2(layer1 []*geojson.Feature,layer2 []*geojson.Feature,keys1 []string,keys2 []string) bool {
	ch := make(chan bool, 2)
	go check2(layer1,layer2,keys1,keys2, ch)

	time.AfterFunc(time.Second*45, func() { ch <- false })
	return <-ch
}


// creates an entirely new feature map
func Create_Layermap(layer1 Layer,layer2 Layer) []*geojson.Feature {
	total := []*geojson.Feature{}

	// creating the new struct that is more lean and 
	// can be sent into a go routine

	var sema2 = make(chan struct{}, 2000)
	fmt.Println()
    mapthing := make([]Layer,len(layer1.BBs))
    cc := make(chan Layer)
	for ii := range layer1.BBs {

		feat := layer1.Features[ii]
		bb := layer1.BBs[ii]
		polygon := layer1.Polygons[ii]
		go func(feat *geojson.Feature,bb m.Extrema,polygon pc.Polygon,cc chan Layer) {

			sema2 <- struct{}{}        // acquire token
			defer func() { <-sema2 }() // release token

			inds := Create_BB_Feats(feat,bb,layer2)
			
			// creating newlaeyr
			newlayer := Layer{BBs:[]m.Extrema{bb},Features:[]*geojson.Feature{feat},Polygons:[]pc.Polygon{polygon}}
			for _,i := range inds {
				newlayer.BBs = append(newlayer.BBs,layer2.BBs[i])
				newlayer.Features = append(newlayer.Features,layer2.Features[i])
				newlayer.Polygons = append(newlayer.Polygons,layer2.Polygons[i])

			}
			cc <- newlayer 
		}(feat,bb,polygon,cc)


		//mapthing[ii] = newlayer
	}

	count := 0
	for range layer1.BBs {
		mapthing[count] = <-cc
		count += 1
		fmt.Printf("\r Mappingshit [%d/%d]",count,len(layer1.BBs))
	}

	fmt.Println()


	// creating channel and sending output
	c := make(chan []*geojson.Feature,len(mapthing))
	var sema = make(chan struct{}, 200)
	unreached := 0
	for _,i := range mapthing {
		go func(i Layer,c chan []*geojson.Feature) {
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			c <- Make_const_polygons(i)

			//c <- Make_const_polygons(i)
		}(i,c)
	}	

	// collecting output from channel
	for i := range mapthing {
		total = append(total,<-c...)
		fmt.Printf("\r Creating Intersecting Features [%d/%d]",i,len(mapthing))
	}
	fmt.Println("\n",unreached,"\n")

	fmt.Print("\n\n")
	return total
}


// creates the string section
func Make_String_Section(newfeats []geojson.Feature) string {
	// creating feature from pointers
	newfeats2 := make([]*geojson.Feature,len(newfeats))
	for ii,i := range newfeats {
		newfeats2[ii] = &geojson.Feature{Geometry:i.Geometry,Properties:i.Properties}
	}

	// creatting string
	fc := &geojson.FeatureCollection{Features:newfeats2}
	shit2,_ := fc.MarshalJSON()
	shit2str := string(shit2[40:len(shit2)-2])
	
	return shit2str
}

// makes sure the smaller layer is always layer1
// this makes the bounding box alg more effecient.
func Fix_Layers(layer1 Layer,layer2 Layer) (Layer,Layer) {
	if len(layer1.BBs) > len(layer2.BBs) {
		dummy := layer2 
		layer2 = layer1
		layer1 = dummy
	}
	return layer1,layer2
}


// given two layer data structures creates a new set of geojson features from 
// the output i.e. a new combined layer
func Combine_Layers(layer1 Layer,layer2 Layer,difference_bool bool) *geojson.FeatureCollection {
	// fixing layers so that layer1 is always the smalller layer
	layer1,layer2 = Fix_Layers(layer1,layer2)

	// creating the intersection part
	feat := Create_Layermap(layer1,layer2)

	// getting teh keys in each layer
	var keys1,keys2 []string
	for k := range layer1.Features[0].Properties {
		keys1 = append(keys1,k)
	}
	for k := range layer2.Features[0].Properties {
		keys2 = append(keys2,k)
	}

	// mapping each layer
	if difference_bool == true {

		layer1map := Map_Layer(layer1.Features,10)
		fmt.Println("\nlayer1 map complete")

		layer2map := Map_Layer(layer2.Features,10)
		fmt.Println("\nlayer2 map complete")
		
		// finally gettting teh differences between each layer
		feat = append(feat,Make_Differences(layer1map,layer2map,keys1,keys2)...)
		fmt.Print("herea.")
	}
	return &geojson.FeatureCollection{Features:feat}

}