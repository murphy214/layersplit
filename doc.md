# Use Case 

This package is designed to take two (nonoverlapping) layers from a geojson file or structure and combine them to create all the uniquely possible polygons from the two layers while combining each feature, with a prefix in front of each field if so desired. This helps in creatng complex polygon hierarchies that are useful more many things, although my main goal with this project is to able to just sit another layer of a hiearchy on top of an existing one and add to without ever needing to worry about it. So in the end I can have a single layer that may represent 100s of layers by combining on top of one another. From here the goal is to create a polygon index which in point in polygon can be solved flat for all layers not just one, or relying on a hierarchy to drill farther darn. The main sticking point I had with building this algorithm for a while was abstracting away the differences between each polygon layer, but I think I've finally built a way to do it pretty effeciently. (see below)

# Algorithm Design 

This algorithm has two main parts being:

	* Layer Intersection
	* Layer Difference 

# Layer Intersection 

The layer intersection part of the algorithm is pretty simple, iterate through the smaller layer sending the bigger layer into a mapping function, collect the possible polygon intersections for each polygon in the small layer. It would look something like this in psuedo-code: 

```
for polygon in smaller_layer:
	possible_polygons = filter_by_boundingbox(polygon,biggerlayer)
```

Of course the code above is mapped and collected in channel, from here we can send into the real compute, function being the actual clipping process. 