package layersplit

import (
	"encoding/csv"
	"fmt"
	pc "github.com/murphy214/polyclip"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// Point represents a point in space.
type Config struct {
	Output   string    // can be "csv" or "slice" currently
	OutFile  string    // the output filename
	Csv1     string    // csv file of layer to combine
	Csv2     string    // csv file of layer to combine
	Layer1   []Polygon // layer to combine if no csv given
	Layer2   []Polygon // layer to combine if no csv given
	Progress int       // steps to increment or show progress creating each new layer
}

// wrapper function for stock csv files to remove boiler plate bullshit
// helper function for reading in ccsvs
func Get_Csv(filename string) [][]string {
	b, err := ioutil.ReadFile(filename) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	str := string(b)
	r := csv.NewReader(strings.NewReader(str))

	// Read all records.
	data, _ := r.ReadAll()
	return data[1:]

}

// making the string that corresponds to each polygon
func Make_Each_Polygon(polygon pc.Polygon) string {
	// making contour string
	contourlist := []string{}
	for _, cont := range polygon {
		ptlist := []string{}
		for _, pt := range cont {
			ptlist = append(ptlist, fmt.Sprintf("[%f,%f]", pt.X, pt.Y))
		}
		contourlist = append(contourlist, fmt.Sprintf("[%s]", strings.Join(ptlist, ",")))
	}
	return fmt.Sprintf("[%s]", strings.Join(contourlist, ","))
}

// for a list of given polygons creates a polygon string
func Make_Polygon_String(poly []Polygon) string {
	polygonstringlist := []string{}
	for _, pp := range poly {
		// making json string from layermap
		newlist := []string{}
		mymap := pp.Layers
		//fmt.Print(pp, "\n")

		for k, v := range mymap {
			if k == "Combined" {
				newlist = append(newlist, v[1:len(v)-1])
			} else {
				newlist = append(newlist, fmt.Sprintf(`'%s':'%s'`, k, v))
			}
		}
		jsonstring := fmt.Sprintf("{%s}", strings.Join(newlist, ","))
		polygonstring := Make_Each_Polygon(pp.Polygon)
		polystring := fmt.Sprintf(`"%s","%s"`, jsonstring, polygonstring)
		polygonstringlist = append(polygonstringlist, polystring)
	}
	return strings.Join(polygonstringlist, "\n")
}

// Given two slices containg two different geographic layers
// returns 1 slice representing the two layers in which there are split into
// smaller polygons for creating hiearchies e.g. slicing zip codes about counties
func Combine_Layers_Slice(layer1 []Polygon, layer2 []Polygon, progress int) []Polygon {
	c := make(chan Output_Struct)
	newlist := []Polygon{}
	for _, row := range layer1 {
		go func(row Polygon, layer2 []Polygon, c chan<- Output_Struct) {
			a := Make_layer_polygon(row, layer2, true)
			//fmt.Print(a, "\n")
			c <- a
		}(row, layer2, c)
	}
	count := 0
	total := 0
	for range layer1 {
		select {
		case msg1 := <-c:
			if count == progress {
				count = 0
				total += progress
				fmt.Printf("[%d/%d]\n", total, len(layer1))
				//fmt.Print(total, "\n")
			}
			//for _, pol := range msg1 {
			//	fmt.Print(pol.Layers, "\n")
			//}
			newlist = append(newlist, msg1.Polylist...)
			count += 1

		}
	}
	return newlist
}

// Given two slices containg two different geographic layers
// returns 1 slice representing the two layers in which there are split into
// smaller polygons for creating hiearchies e.g. slicing zip codes about counties
func Combine_Layers_Csv(layer1 []Polygon, layer2 []Polygon, outfilename string, progress int) {
	// creating channel
	c := make(chan Output_Struct)

	// creating file
	ff, _ := os.Create(outfilename)
	ff.WriteString("LAYERS,COORDS")
	for _, row := range layer1 {
		go func(row Polygon, layer2 []Polygon, c chan<- Output_Struct) {
			a := Make_layer_polygon(row, layer2, false)
			c <- a
		}(row, layer2, c)
	}

	// iterating through each recieved channel output
	count := 0
	total := 0
	for range layer1 {
		select {
		case msg1 := <-c:
			if count == progress {
				count = 0
				total += progress
				fmt.Printf("[%d/%d]\n", total, len(layer1))

			}
			ff.WriteString("\n" + msg1.Polystring)
			count += 1

		}
	}

}

// given a configuration struct combines two layers
// see configuration structure on line 9
func Combine_Layers(configs Config) []Polygon {
	// going through the configuration
	layers := [][]Polygon{}
	if len(configs.Csv1) != 0 {
		layer := Make_Layer(Get_Csv(configs.Csv1), strings.Split(configs.Csv1, ".")[0])
		layers = append(layers, layer)
	}
	if len(configs.Csv2) != 0 {
		layer := Make_Layer(Get_Csv(configs.Csv2), strings.Split(configs.Csv2, ".")[0])
		layers = append(layers, layer)
	}
	if len(configs.Layer1) != 0 {
		layers = append(layers, configs.Layer1)
	}
	if len(configs.Layer2) != 0 {
		layers = append(layers, configs.Layer2)
	}

	// evaluating layers
	var layer1, layer2 []Polygon
	if len(layers[0]) > len(layers[1]) {
		layer1 = layers[1]
		layer2 = layers[0]
	} else {
		layer1 = layers[0]
		layer2 = layers[1]
	}

	// setting default progress increment if none given
	if len(strconv.Itoa(configs.Progress)) == 1 {
		configs.Progress = 100
	}

	// evaulating the two different types of output
	outfilename := ""
	if configs.Output == "csv" {
		// evaluating outfilename
		if (len(configs.OutFile) == 0) && (configs.Output == "csv") {
			outfilename = "results.csv"
		} else if configs.Output == "csv" {
			outfilename = configs.OutFile
		}
		Combine_Layers_Csv(layer1, layer2, outfilename, configs.Progress)
	} else if configs.Output == "layer" {
		layer := Combine_Layers_Slice(layer1, layer2, configs.Progress)
		newlist := []Polygon{}
		for _, poly := range layer {
			mymap := poly.Layers
			newlist2 := []string{}
			for k, v := range mymap {

				newlist2 = append(newlist2, fmt.Sprintf(`'%s':'%s'`, k, v))
			}
			jsonstring := fmt.Sprintf("{%s}", strings.Join(newlist2, ","))
			poly.Area = jsonstring
			poly.Layer = "Combined"
			newlist = append(newlist, poly)
		}
		return newlist
	}
	return []Polygon{}

}
