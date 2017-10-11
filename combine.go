package layersplit 

import (
	pc "github.com/murphy214/polyclip"
	//"fmt"
	//"time"
	m "github.com/murphy214/mercantile"
	"sort"
	"math/rand"

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
	if len(layer) == 1 {
		return layer[0]
	}

	poly := layer[0]
	for _,i := range layer[1:] {
		poly = poly.Construct(pc.UNION,i)
	}
	return poly
}

type reverseSort struct { 
        sort.Interface 
} 

func (r reverseSort) Less(i,j int) bool { 
        return r.Interface.Less(j,i) 
} 

func Reverse(x sort.Interface) sort.Interface { 
        return reverseSort{x} 
} 
// input6 is a slice of output features
func Make_Big3(layer []pc.Polygon) pc.Polygon {
	if len(layer) == 1 {
		return layer[0]
	}

	areamap := map[float64]pc.Polygon{}
	keys := []float64{}
	for _,poly := range layer {
		if len(poly) > 0 {
			if len(poly[0]) > 1 {			
				bb := poly.BoundingBox()
				newbb := m.Extrema{W:bb.Min.X,E:bb.Max.X,S:bb.Min.Y,N:bb.Max.Y}
				area := AreaBds(newbb)
				areamap[area] = poly
				keys = append(keys,area)
			}
		}
	}

	sort.Sort(Reverse(sort.Float64Slice(keys[:]))) 
	//fmt.Println("\n",keys,"\n")

	newlayer := []pc.Polygon{}
	for _,k := range keys {
		newlayer = append(newlayer,areamap[k])
	}
	layer = newlayer



	if len(layer) == 0 {
		return pc.Polygon{}
	}

	poly := layer[0]
	for _,i := range layer[1:] {
		poly = poly.Construct(pc.UNION,i)
	}
	return poly
}



// makes the total big polygon of a given layer recursively
func Make_Total_Big(layer []pc.Polygon) pc.Polygon {
	zoom := 14
	newlayer := layer
	for len(newlayer) > 4 {
		newlayer = Make_Run(newlayer,zoom)
		//fmt.Printf("Creating Union Polygon Size: %d, Zoom: %d\n",len(newlayer),zoom)
		zoom = zoom - 1
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

// gets a random point
func Random_Point(pt pc.Point) pc.Point {
	sign1 := rand.Intn(2)
	factor1 := float64(rand.Intn(1000)) * .0000000001
	if sign1 == 1 {
		factor1 = factor1 * -1
	}
	sign2 := rand.Intn(2)
	factor2 := float64(rand.Intn(1000)) * .0000000001
	if sign2 == 1 {
		factor2 = factor2 * -1
	}

	return pc.Point{X:pt.X+factor1,Y:pt.Y+factor2}
}