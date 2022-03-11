# Go vs GPU: Fast Spatial Joins

I read [a post yesterday](https://medium.com/swlh/how-to-perform-fast-and-powerful-geospatial-data-analysis-with-gpu-48f16a168b10) that demonstrates using the GPU to process a spatial join on bigdata.
The author takes a recordset of 9 million parking violations in Philadelphia and spatially joins them to a recordset of 150 neighborhood polygons.
He uses the GPU libraries [rapidsai/cuDF](https://github.com/rapidsai/cudf) to read the data and [rapidsai/cuSpatial](https://github.com/rapidsai/cuspatial) to do spatial point-in-polygon operations.

His results are very good. Reading the input data takes 4 seconds and performing the spatial join takes only 13 seconds.

I was curious to see how well a Go program using the CPU would compare, so I wrote this little program.

I used the Go libraries [tidwall/geojson](https://github.com/tidwall/geojson) for storing the neighborhood polygons and [tidwall/rtree](https://github.com/tidwall/rtree) for the spatial index. These are the same foundational libraries I use in [Tile38](https://github.com/tidwall/tile38).


## Results

On my 2019 Macbook (2.4 GHz 8-Core Intel Core i9):

```
Loading neighborhoods... 0.01 secs
Loading violations... 3.90 secs
Joining neighborhoods and violations... 2.75 secs
Writing output... 0.46 secs
Total execution time... 7.12 secs
```

Most of the time is taken up reading the violations CSV file.

The `Joining neighborhoods and violations` operation is where the point-in-polygon and spatial join operation happens.

## Downloading the data

All data is download to the `data` directory.
The `ogr2ogr` command is provided by [GDAL](https://gdal.org).

```sh
# Create the directories for storing data
mkdir -p data/shapes

# Download the parking violations, this will take awhile.
curl "https://phl.carto.com/api/v2/sql?filename=parking_violations&format=csv&skipfields=cartodb_id,the_geom,the_geom_webmercator&q=SELECT%20*%20FROM%20parking_violations%20WHERE%20issue_datetime%20%3E=%20%272012-01-01%27%20AND%20issue_datetime%20%3C%20%272017-12-31%27" > data/phl_parking.csv

# Download the Philadelphia neighborhoods shapes and unzip it.
wget -P data/shapes https://github.com/azavea/geo-data/raw/master/Neighborhoods_Philadelphia/Neighborhoods_Philadelphia.zip 

unzip -d data/shapes data/shapes/Neighborhoods_Philadelphia.zip

# Convert the neighborhood shapes into wgs84 GeoJSON
ogr2ogr -t_srs EPSG:4326 data/Neighborhoods_Philadelphia.json data/shapes/Neighborhoods_Philadelphia.shp
```

## Running 

```sh
git clone https://github.com/tidwall/fast-spatial-joins
cd fast-spatial-joins
go run main.go
```

The final output is written to `data/output.csv`.

Here's a snapshot of what the output looks like:

```
anon_ticket_number,neighborhood
1777797,Center City East
1777798,Chinatown
1777799,Wister
1777801,Center City East
1777802,Logan Square
1777803,Old City
1777804,Northern Liberties
1777805,Newbold
1777806,Old City
1777808,Society Hill
1777809,Spring Garden
1777810,Rittenhouse
1777811,Logan Square
1777813,Graduate Hospital
1777814,Point Breeze
1777815,Washington Square West
1777816,Rittenhouse
1777817,Wister
1777818,Society Hill
1777820,Logan Square
```
