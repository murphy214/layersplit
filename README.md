# layersplit

**NOTE: I need to update this read me, to lazy to do it now but have completely rewritten this repo, this example is useless lol.**

Layer split abstractions for splitting up multiple GIS polygon layers into smaller constituent polygons.

This project takes a [modified polyclip library]() and uses it to take entire polygon layers (eg. counties,zips,states) and build a hierarchy / smallest polygons from them. This is useful because often time polygon sets that you think would be completely related are sometimes not so related when expecting perfect puzzle like hierarchy. In other words, zipcodes strattle county lines,often times split evenly between them, zipcodes often times don't exist in some disparate places. So this project tries to divide the polygons of different layers into unique polygons representitve of a layer set i.e. a polygon represents a part of a zipcode, a part of a county and part of a state, all while maintaining fields. 

However there is one major caveat to this so far, currently I have no reliable way of getting the inverse of polygon, where you have a top level polygon and other constituent polygons within the top level representing where holes are in the top level set that are occupied by another polygon, containing both layers, but if the top level polygon isn't completely filled I have no way of retrieving the unoccupied top level polygon that doesn't intersect with the other representitive layer. While what I have can represent the same polygon I want in most software, it doesn't play nice with the intersection algorithm.

Other than it works pretty decent API is alright so far.

# Issues 
The biggest issue that currently exists with this module is negative space. Meaning, I need to describe a polygon that I generally use complex holes to fill around the missing geometry space within a polygon. However, the way this linesplit algorithm is being used later on the down the pipeline is for creating polygon indexs, and having a polygon(s) represented by all the other polygons already drawn isn't ideal for how the index is created.

It needs to be represented by a set or simple polygons representing the exact the polygon in question not the geometric equivalent of the polygon. (i.e. a square with x number of holes leaving only the polygon left to be drawn)

This geometry representing negative space is pretty integral to how this algorithm is suppose to work. Imagine something like a base tile canvas and for every layer you add you divide the polygons within that layer to represent each unique set of polygon type that may occur. If this is done correctly you can build an entire point in polygon layer from every single layer you want to know information about. (i.e. one point in polygon alg. gets county,state,congressional district, etc. etc.) More importantly utilizing the negatize space not all polygons have to be within a pure hiearchy its built as its divided up.


# Example 
```go
package main

import (
	l "layersplit"
)

func main() {
	// reading in csvs files NOTE: this doesn't always have to be done
	// reading in county csv layer struct
	ct := l.Get_Csv("county.csv")
	ctt := l.Make_Layer(ct, "COUNTY")

	// reading in zip csv layer struct
	zt := l.Get_Csv("zip.csv")
	ztt := l.Make_Layer(zt, "ZIP")

	// reading in state csv layer struct
	st := l.Get_Csv("states.csv")
	stt := l.Make_Layer(st, "STATES")

	// getting the configuration struct for combining states layer and counties
	configs := l.Config{Output: "layer", Layer1: stt, Layer2: ctt}

	// from the config struct we send in our arguments and
	// a new layer struct is returned
	layer := l.Combine_Layers(configs)

	// now using the newly created layer and the last layer to create our final output
	// this will be written two a csv file
	// A more cohesive processing struct or logic will probably be added later.
	configs = l.Config{Output: "csv", Layer1: l.Combine_Layers(configs), Layer2: ztt}

	// finally executing the final combine function that outputs a csv file
	l.Combine_Layers(configs)
}

```


#### Pictures
![](https://user-images.githubusercontent.com/10904982/27519281-42e6b714-59be-11e7-9a60-4a897a99955a.png)
![](https://user-images.githubusercontent.com/10904982/27519282-42ef7a02-59be-11e7-9131-f03e0fd66b28.png)
