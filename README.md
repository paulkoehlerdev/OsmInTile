# OsmInTile

OsmInTile is an experimental Go-based project for generating indoor vector tiles using OpenStreetMap (OSM) data. This tool is designed to parse OSM data and produce vector tiles suitable for indoor mapping applications, facilitating easier visualization and interaction with indoor spaces.

> [!CAUTION]
> This project is a work in progress.
> See [Sec. TODO](TODO) to see the progress.

# Table of Contents

1. Introduction
1. TODOs
1. Installation
1. Contributing
1. License
1. Acknowledgements

## Introduction

Indoor mapping presents unique challenges compared to outdoor mapping, primarily due to the complexity and variability of indoor spaces. OsmInTile aims to simplify this by providing a tool that extracts indoor-related data from OSM and transforms it into vector tiles, which are ideal for high-performance rendering in applications like web maps and mobile apps.

### TODOs

- OSM Data Parsing: 
  - [x] Parsing osm `.pbf` files
  - [x] Parsing osm `.osm`-XML files
  - [x] Filtering unnecessary features on the fly
  - [x] Insert important information into [Spatialite](https://www.gaia-gis.it/fossil/libspatialite/index)
  - [ ] Optimize inserted data more by filtering unused tags
- Vector Tile Generation:
  - [x] Get Vector tiles with basic Room information to the user
  - [ ] Create Sprites and Texts
- Customizable Output:
  - [ ] Allow customization of tile properties such as zoom levels, feature selection, and more
  - [ ] Allow customization of map styles
  - [ ] Make Demo Frontend removable
- High Performance:
  - [ ] Create a on-disk and in-memory caching layer to limit requests to sqlite
  - [ ] Prerender adjacent tiles in spare time
  - [ ] Allow for rate-limits
- Demo Frontend:
  - [x] Create a basic Demo frontend with [MapLibre](https://maplibre.org/)
  - [ ] Create level selection
  - [ ] Create Documentation Page for using the indoor map in your project
- Documentation and Distribution:
  - [ ] Document the usage and configuration
  - [ ] Create Prebuild binaries
  - [ ] Distribute via apt and docker hub

## Installation

### Prerequisites

You will need to have docker installed and running on your system

### Installation Steps

1. Download your data from Geofabrik in either the `.osm` or `.pbf` format (you may also cut it with [osmium](https://osmcode.org/osmium-tool/))
2. Start the Server with `make run --osm-file <path to your file>`
3. For more information see the help command or code `make help`

## Contributing

We welcome contributions from the community! 

To contribute:

1. Fork the repository.
1. Create a new branch (`git checkout -b feature/your-feature`).
1. Commit your changes (`git commit -am 'Add some feature'`).
1. Push to the branch (`git push origin feature/your-feature`).
1. Create a new Pull Request.

Please ensure your code follows the Go code style guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgements

**OpenStreetMap:** For providing the open data that makes this project possible.

**[@paulmach](https://github.com/paulmach):** For providing great geodata libraries like [paulmach/orb](https://github.com/paulmach/orb) and [paulmach/osm](https://github.com/paulmach/osm)

**Go:** For being a powerful and efficient programming language.