package layersplit


import (
	"github.com/paulmach/go.geojson"
	pc "github.com/murphy214/polyclip"
	m "github.com/murphy214/mercantile"
	"fmt"
	//"time"
	"math/rand"
)

// structure for creaitng output
type Output_Feature struct {
	Polygon pc.Polygon
	Feature geojson.Feature
	BB m.Extrema
}

// structure for a map output
type Map_Output struct {
	Key Output_Feature
	Feats []Output_Feature
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
				thing2.Add(pc.Point{X: i[0], Y: i[1]})
			}
		}
		things.Add(thing2)
	}

	return things
}


// creates a structure that will be representitive of a layer
func Make_Structs(gjson *geojson.FeatureCollection,prefix string) []Output_Feature {
	gjson = Split_Multi(gjson)
	outputs := []Output_Feature{}
	c := make(chan Output_Feature)
	for _,i := range gjson.Features {
		go func(i *geojson.Feature,c chan Output_Feature) {
			if prefix != "NONE" {
				i.Properties = Add_Prefix(i.Properties,prefix)
			}
			feat := Output_Feature{Polygon:Make_Polygon(i.Geometry.Polygon),Feature:*i}
			bb := feat.Polygon.BoundingBox()
			feat.BB = m.Extrema{W:bb.Min.X,E:bb.Max.X,S:bb.Min.Y,N:bb.Max.Y}
			c <- feat
		}(i,c)

	}

	for range gjson.Features {
		val := <- c
		if len(val.Polygon) > 0 {
			outputs = append(outputs,val)

		}
	}

	return outputs
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
func (feat Output_Feature) Intersect(bds m.Extrema) bool {
	bdsref := feat.BB
	if Overlapping_1D(bdsref.W-.0000001,bdsref.E+.0000001,bds.W-.0000001,bds.E+.0000001) && Overlapping_1D(bdsref.S-.0000001,bdsref.N+.0000001,bds.S-.0000001,bds.N+.0000001) {
		return true
	} else {
		return false
	}

	return false
}

// creates bounding box featues
func Create_BB_Feats(feat Output_Feature,layer2 []Output_Feature) ([]Output_Feature) {
	bb := feat.BB
	feats := []Output_Feature{}
	for _,i := range layer2 {
		boolval := i.Intersect(bb)
		if boolval == true {
			feats = append(feats,i)
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

// taking the found polygons and returning a list of each intersected polygon
func Make_const_polygons(first Output_Feature,finds []Output_Feature) []geojson.Feature {
	feats := []geojson.Feature{}
	for _, i := range finds {
		result := first.Polygon.Construct(pc.INTERSECTION, i.Polygon)
		// adding the the result to newlist if possible
		if len(result) != 0 {

			mymap := Combine_Properties(first.Feature.Properties,i.Feature.Properties)
 			polygons := Lint_Polygons(result)
			for _,polygon := range polygons {
				feats = append(feats,geojson.Feature{Properties:mymap,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"}})
			}
		} 

	}
	return feats
}


// creates an entirely new feature map
func Create_Layermap(layer1 []Output_Feature,layer2 []Output_Feature) []geojson.Feature {
	total := []geojson.Feature{}

	// creating the new struct that is more lean and 
	// can be sent into a go routine
    mapthing := make([]Map_Output,len(layer1))
	for ii,i := range layer1 {
		feats := Create_BB_Feats(i,layer2)
		mapthing[ii] = Map_Output{Key:i,Feats:feats}
	}

	// creating channel and sending output
	c := make(chan []geojson.Feature,len(mapthing))
	var sema = make(chan struct{}, 1000)
	for _,i := range mapthing {
		go func(i Map_Output,c chan []geojson.Feature) {
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			c <- Make_const_polygons(i.Key,i.Feats)
		}(i,c)
	}

	// collecting output from channel
	for i := range mapthing {
		total = append(total,<-c...)
		fmt.Printf("\r[%d/%d]",i,len(mapthing))
	}
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

// top level combine function
func Combine_Layers(layer1 []Output_Feature,layer2 []Output_Feature) []*geojson.Feature {
	// linting the size of each layer
	if len(layer1) > len(layer2) {
		dummylayer := layer2
		layer2 = layer1
		layer1 = dummylayer
	}
	
	// creating new features
	fmt.Print("Starting Layer Intersections\n")

	// getting teh first set of intersecting values
	// due to interfence with pointers currently writes to string
	// this could be removed later
	layer22 := layer2

	newfeats := Create_Layermap(layer1,layer2)
	str1 := Make_String_Section(newfeats)
	fmt.Print(newfeats[0],"imherea\n\n")
	// getting the different values for each layer
	newfeats = Make_Big_Both(layer1,layer22)
	str2 := Make_String_Section(newfeats)

	// combining each string list
	str1 = "[" + str1 + "," + str2 + "]"

	// parsing back into geojson struct
	fc,err := geojson.UnmarshalFeatureCollection([]byte(`{"type": "FeatureCollection", "features":` + str1 + "}"))
	if err != nil {
		fmt.Print(err,"\n")
	}
	return fc.Features
}