# layersplit
Layer split abstractions for splitting up multiple GIS polygon layers into smaller constituent polygons.

This project takes a [modified polyclip library]() and uses it to take entire polygon layers (eg. counties,zips,states) and build a hierarchy / smallest polygons from them. This is useful because often time polygon sets that you think would be completely related are sometimes not so related when expecting perfect puzzle like hierarchy. In other words, zipcodes strattle county lines,often times split evenly between them, zipcodes often times don't exist in some disparate places. So this project tries to divide the polygons of different layers into unique polygons representitve of a layer set i.e. a polygon represents a part of a zipcode, a part of a county and part of a state, all while maintaining fields. 

However there is one major caveat to this so far, currently I have no reliable way of getting the inverse of polygon, where you have a top level polygon and other constituent polygons within the top level representing where holes are in the top level set that are occupied by another polygon, containing both layers, but if the top level polygon isn't completely filled I have no way of retrieving the unoccupied top level polygon that doesn't intersect with the other representitive layer. While what I have can represent the same polygon I want in most software, it doesn't play nice with the intersection algorithm.

Other than it works pretty decent API is alright so far.
