package layersplit 

import (
	pc "github.com/murphy214/polyclip"
	"fmt"
	//"time"
	"github.com/paulmach/go.geojson"
	m "github.com/murphy214/mercantile"

)

// creating tilemap
func Make_Run(layer []pc.Polygon,zoom int) []pc.Polygon {
	totalmap := map[m.TileID][]pc.Polygon{}
	for _,i := range layer {
		firstpt := i[0][0]
		tileid := m.Tile(firstpt.X,firstpt.Y,zoom)
		totalmap[tileid] = append(totalmap[tileid],i)
	}

	c := make(chan pc.Polygon,len(totalmap)) 
	for _,v := range totalmap {
		go func(v []pc.Polygon,c chan pc.Polygon) {
			c <- Make_Big(v)
		}(v,c)
	}

	totaloutput := []pc.Polygon{}
	for range totalmap {
		feat := <-c
		if len(feat) > 0 {
			if len(feat[0]) > 0 {
				totaloutput = append(totaloutput,feat)
			}
		}
	}

	return totaloutput
}


// input is a slice of polygons
func Make_Big2(layer []pc.Polygon) pc.Polygon {
	poly := layer[0]
	for _,i := range layer[1:] {
		poly = poly.Construct(pc.UNION,i)
	}
	return poly
}

// input6 is a slice of output features
func Make_Big(layer []pc.Polygon) pc.Polygon {
	poly := layer[0]
	for _,i := range layer[1:] {
		poly = poly.Construct(pc.UNION,i)
	}
	return poly
}

// makes the total big polygon of a given layer recursively
func Make_Total_Big(layer []pc.Polygon) pc.Polygon {
	zoom := 12
	newlayer := layer
	for len(newlayer) > 100 {
		newlayer = Make_Run(newlayer,zoom)
		fmt.Printf("Creating Union Polygon Size: %d, Zoom: %d\n",len(newlayer),zoom)
		zoom = zoom - 2
	}

	return Make_Big(newlayer)
}

// makign empty properties
func Make_Empty_Properties(props1 map[string]interface{},props2 map[string]interface{}) (map[string]interface{},map[string]interface{}) {
	for k := range props1 {
		props1[k] = "NONE"
	}
	for k := range props2 {
		props2[k] = "NONE"
	}
	return props1,props2
}

// getting layer size and sema size
func Get_Sema_Size(sizelayer int) int {
	semasize := sizelayer / 25
	return semasize
}


// given a layer and a total union polygon gets the given difference of the layer
// so taht properties can be maintained for each specific polygon
func Difference_Layer(layer []Output_Feature,poly pc.Polygon,mymap map[string]interface{}) []geojson.Feature {
	totalfeats := []geojson.Feature{}
	size := len(layer)
	c := make(chan []geojson.Feature,len(layer))
	semasize := Get_Sema_Size(size)
	var sema = make(chan struct{}, semasize)
	for _,i := range layer {
		go func(i Output_Feature,c chan []geojson.Feature) {	
			sema <- struct{}{}        // acquire token
			defer func() { <-sema }() // release token

			feats := []geojson.Feature{}	
			result := i.Polygon.Construct(pc.DIFFERENCE,poly)
			polygons := Lint_Polygons(result)
			newmap := Combine_Properties(i.Feature.Properties,mymap)
			for _,polygon := range polygons {
				feats = append(feats,geojson.Feature{Properties:newmap,Geometry:&geojson.Geometry{Polygon:Convert_Float(polygon),Type:"Polygon"}})
			}
			c <- feats
		}(i,c)

	}

	// collecting features 
	count := 0
	total := 0
	for range layer {
		totalfeats = append(totalfeats, <-c...)
		count += 1
		if count == semasize {
			total += semasize
			count = 0
			fmt.Printf("[%d/%d]\n",total,size)

		}

	}
	return totalfeats
}

