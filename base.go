package layersplit

import (
	"github.com/paulmach/go.geojson"
	pc "github.com/murphy214/polyclip"
	m "github.com/murphy214/mercantile"
	"fmt"
	"math/rand"
	g "github.com/murphy214/geobuf"
	"io/ioutil"
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
	aa := &geojson.Feature{Properties:mymap,Geometry:geometry,ID:a.ID}
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
				thing2.Add(pc.Point{X: i[0], Y: i[1]})
			}
		}
		if len(thing2) >= 2 {
			things.Add(thing2)
		}
	}
	return things
}

// creates a pc polygon
func Make_Polygon2(coords [][][]float64) pc.Polygon {
	thing2 := pc.Contour{}
	things := pc.Polygon{}
	for _, coord := range coords {
		thing2 = pc.Contour{}

		for _, i := range coord {
			if len(i) >= 2 {
				// moving sign in 10 ** -7 pla
				sign1 := rand.Intn(1)
				factor1 := float64(rand.Intn(100)) * .0000000001
				if sign1 == 1 {
					factor1 = factor1 * -1
				}
				sign2 := rand.Intn(1)
				factor2 := float64(rand.Intn(100)) * .0000000001
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

// reads a geojson file and fuzzes the coordinate values so they wont interfere with others
func Read_Geojson(filename string) *geojson.FeatureCollection {	
	e, _ := ioutil.ReadFile(filename)
	fc, _ := geojson.UnmarshalFeatureCollection(e)
	return Make_Fuzz(fc)
}

// fuzzes the input feature collection
func Make_Fuzz(fc *geojson.FeatureCollection) *geojson.FeatureCollection {
	for ii,i := range fc.Features {
		fc.Features[ii].Geometry.Polygon = Convert_Float(Make_Polygon2(i.Geometry.Polygon))
	}
	return fc
}

// creates a structure that will be representitive of a layer
func Make_Structs_Fuzz(gjson []*geojson.Feature,fuzzbool bool) Layer {
	c := make(chan Output_Feature)
	for _,i := range gjson {
		go func(i *geojson.Feature,c chan Output_Feature) {
			var feat Output_Feature
			poly := Make_Polygon_Round(i.Geometry.Polygon)
			if fuzzbool == true {
				feat = Output_Feature{Polygon:poly,Feature:i}
			} else {
				feat = Output_Feature{Polygon:poly,Feature:i}
			}
			bb := poly.BoundingBox()
			feat.BB = m.Extrema{W:bb.Min.X,E:bb.Max.X,S:bb.Min.Y,N:bb.Max.Y}
			c <- feat
		}(i,c)

	}
	bbs := []m.Extrema{}
	feats := []*geojson.Feature{}
	polygons := []pc.Polygon{}
	for range gjson {
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

func Ensure_Round_Polygon(coords [][][]float64) [][][]float64  {
	new_polygon := [][][]float64{}
	for _,cont := range coords {
		new_contour := [][]float64{}
		for _,point := range cont { 
			point[0] = Round(point[0],.5,6)
			point[1] = Round(point[1],.5,6)
			new_contour = append(new_contour,point)
		}
		new_polygon = append(new_polygon,new_contour)
	}
	return new_polygon
}



// creates a structure that will be representitive of a layer
func Make_Structs(gjson *geojson.FeatureCollection,prefix string) []*geojson.Feature {
	gjson = Split_Multi(gjson)
	newlist := []*geojson.Feature{}
	for ii,i := range gjson.Features {
		i.ID = ii
		if prefix != "NONE" {
			i.Properties = Add_Prefix(i.Properties,prefix)
		}
		i.Geometry.Polygon = Ensure_Round_Polygon(i.Geometry.Polygon)
		newlist = append(newlist,i)
	}

	return newlist
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
	if Overlapping_1D(bdsref.W,bdsref.E,bds.W,bds.E) && Overlapping_1D(bdsref.S,bdsref.N,bds.S,bds.N) {
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


// writing Write_Sequential shit bruh
func Write_Sequential(feats []*geojson.Feature) []byte {
	bytevals := []byte{}
	for _,feat := range feats {
		bytevals = append(bytevals,g.Write_Feature(feat)...)
	}	
	return bytevals
}


// creates a point map from the "first" polygon
func Make_Point_Map(polygon pc.Polygon) map[pc.Point]string {
	newmap := map[pc.Point]string{}
	for _,cont := range polygon {
		for _,pt := range cont { 
			newmap[pt] = ""
		}
	}
	return newmap
}

// this function covers the rare corner case in which a triangle 
// has one of its point exactly on the same point as a point within first polygon
// we will solve this corner case by randomizing the point that lies on the same alignment.
// the point will be OUTSIDE the outer contour for the reason that if there is 
// is an intersection it will be from another of the 3 points and this point wont matther
func Check_Triangle(polygon pc.Polygon,newmap map[pc.Point]string,outercont pc.Contour) pc.Polygon {	
	newpolygon := pc.Polygon{}
	newcont := pc.Contour{}
	for _,pt := range polygon[0] {
		newcont.Add(pt)
		_,ok := newmap[pt]
		if ok == true {	
			withinbool := false
			var newpt pc.Point
			for withinbool == false {
				newpt = Random_Point(pt)
				withinbool = outercont.Contains(newpt)
			}
			newcont[len(newcont)-1] = newpt
		}
		withinbool := outercont.Contains(pt)
		if withinbool == true {
			newpt := Random_Point(pt)
			newcont.Add(newpt)
		}

	}
	newpolygon.Add(newcont)	
	return newpolygon
}

// Lints a corner case trangle
func Lint_Triangle(polygon pc.Polygon,first pc.Polygon,mymap map[pc.Point]string) pc.Polygon {
	first_contour := first[0]
	totalbool := false
	for _,point := range polygon[0] {

		withinbool := first_contour.Contains(point)
		_,ok := mymap[point]
		if withinbool == true && ok == false {
			totalbool = true
		}
	}
	if totalbool == false {
		polygon = pc.Polygon{}
	} else {
		polygon = Check_Triangle(polygon,mymap,first[0])
	}
	return polygon
}


// taking the found polygons and returning a list of each intersected polygon
func Make_const_polygons(layer Layer) []*geojson.Feature {
	debug := false
	if debug == true {
		ioutil.WriteFile("c.buf",Write_Sequential(layer.Features),0666)
	}

	first := layer.Polygons[0]
	first_feature := layer.Features[0]
	layer.Features = layer.Features[1:]
	layer.BBs = layer.BBs[1:]
	layer.Polygons = layer.Polygons[1:]
	firstmap := Make_Point_Map(first)

	debug2 := false
	//var sema = make(chan struct{}, 1)

	c := make(chan []*geojson.Feature)
	for i, val := range layer.Features {
		go func(i int,val *geojson.Feature,c chan []*geojson.Feature) {
			//sema <- struct{}{}        // acquire token
			//defer func() { <-sema }() // release token

			tempfeats := []*geojson.Feature{}
			newpoly := layer.Polygons[i]
			feat := DeepCopy(val)

			if len(newpoly) == 1 && len(newpoly[0]) == 3 {
				//fmt.Println("shit")
				newpoly = Lint_Triangle(newpoly,first,firstmap)
			}

			// debugging shit
			if debug2 == true {
				fc := &geojson.FeatureCollection{Features:[]*geojson.Feature{val,first_feature}}
				shit,_ := fc.MarshalJSON()
				ioutil.WriteFile("gf/a.geojson",[]byte(shit),0666)
			}

			result := first.Construct(pc.INTERSECTION, newpoly)
			// adding the the result to newlist if possible
			if len(result) != 0 {
				for k,v := range first_feature.Properties {
					feat.Properties[k] = v
				}

	 			polygons := Lint_Polygons(result)
				for _,polygon := range polygons {
					mine := &geojson.Feature{Properties:feat.Properties,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"},ID:feat.ID}				
					tempfeats = append(tempfeats,mine)
				}
			} 
			c <- tempfeats
		}(i,val,c)
	}

	feats := []*geojson.Feature{}
	for range layer.Features {
		//fmt.Println(i,len(layer.Features))
		feats = append(feats,<-c...)
	}

	return feats
}

// creates an entirely new feature map
func Create_Layermap(layer1 Layer,layer2 Layer) []*geojson.Feature {
	total := []*geojson.Feature{}

	// creating the new struct that is more lean and 
	// can be sent into a go routine
	var sema2 = make(chan struct{}, 2000)
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
		fmt.Printf("\rMappingshit [%d/%d]           ",count,len(layer1.BBs))
	}


	// creating channel and sending output
	c := make(chan []*geojson.Feature,len(mapthing))
	var sema = make(chan struct{}, 250)
	for _,i := range mapthing {
		go func(i Layer,c chan []*geojson.Feature) {
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			eh := Make_const_polygons(i)
			//fmt.Println(len(eh))
			if len(eh) > 0 {
				c <- eh
			} else {
				c <- []*geojson.Feature{}
			}
		}(i,c)
	}	

	// collecting output from channel
	for i := range mapthing {
		total = append(total,<-c...)
		fmt.Printf("\rCreating Intersecting Features [%d/%d]           ",i,len(mapthing))
	}
	return total
}


// makes sure the smaller layer is always layer1
// this makes the bounding box alg more effecient.
func Fix_Layers(layer1 []*geojson.Feature,layer2 []*geojson.Feature) ([]*geojson.Feature,[]*geojson.Feature) {
	if len(layer1) > len(layer2) {
		dummy := layer2 
		layer2 = layer1
		layer1 = dummy
	}
	return layer1,layer2
}


// given two layer data structures creates a new set of geojson features from 
// the output i.e. a new combined layer
func Combine_Layers(layer1 []*geojson.Feature,layer2 []*geojson.Feature) []*geojson.Feature {
	// fixing layers so that layer1 is always the smalller layer
	layer1,layer2 = Fix_Layers(layer1,layer2)

	size_before1 := len(layer1)
	size_before2 := len(layer2)

	feats1,feats2 := Make_Differences(layer1,layer2)

	newlayer1 := Make_Structs_Fuzz(layer1,true)
	newlayer2 := Make_Structs_Fuzz(layer2,true)

	// creating the intersection part
	feat := Create_Layermap(newlayer1,newlayer2)
	size_intersection := len(feat)
	feat = append(feat,feats1...)
	feat = append(feat,feats2...)

	fmt.Printf("\nLayer1 Input Size: %d\nLayer2 Input Size: %d\n\tLayer1 Output Difference Size: %d\n\tLayer2 Output Difference Size: %d\n\tIntersection Output Size: %d\nTotal Output Size: %d\n",size_before1,size_before2,len(feats1),len(feats2),size_intersection,len(feat))

	return feat

}



// given two layer data structures creates a new set of geojson features from 
// the output i.e. a new combined layer
func Combine_Layers2(layer1 []*geojson.Feature,layer2 []*geojson.Feature) []*geojson.Feature {
	// fixing layers so that layer1 is always the smalller layer
	layer1,layer2 = Fix_Layers(layer1,layer2)


	feat1,feat2 := Make_Differences(layer1,layer2)

	
	return append(feat1,feat2...)

}
