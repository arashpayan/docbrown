# docbrown
This is a tool for generating documentation for the ISS API from comments directly in the ISS code. For convenience, a binary version of this project is included in this repo.

## Generate documentation
From within this project directory:
```
./docbrown /path/to/iss/code /output/directory/for/documetation
```

## Writing documentation
The documentation presents code based on named packages it finds defined in comments in the code (**not** Go packages). These package names roughly correpond to REST endpoint roots e.g. The _Windows_ package documents the various functionality available at the endpoints that start with `/1/windows`. To define a new package, simply use the package name when documenting a particular endpoint. For example, the documentation for the function that handles the `GET /1/windows` should be placed right above the function that actually performs the functionality in the Go code. Here is an example:
```
/*
@package Windows
@endpoint /windows
@method GET
@purpose Get all windows
@description Returns an array of all the open windows on the server.
@sampleResponse
``` json
[
  {
    "id":1,
    "type":"hdmi",
    "position":{
      "x":0,
      "y":0,
      "width":0.5,
      "height":0.4992389649923897
    },
    "opacity":100,
    "has_audio_focus":false,
    "hdmi_properties":{
      "port":1,
      "volume":50,
      "z_index":0
    }
  },
  {
    "id":2,
    "type":"web",
    "position":{
      "x":0.5,
      "y":0.4992389649923897,
      "width":0.5,
      "height":0.4992389649923897
    },
    "opacity":100,
    "has_audio_focus":false,
    "web_properties":{
      "url":"http://www.google.com/",
      "title":"",
      "is_loading":false,
      "can_go_back":false,
      "can_go_forward":false,
      "z_index":1,
      "zoom_factor":1
    }
  }
]
```​
*/
```

Here is an example of documentation for a web socket command:
```
/*
@package Windows
@command set_zoom_factor
@description Set the zoom amount of the page.
@sampleBody
``` json
{
  "command": set_zoom_factor,
  "window_id": <integer window id>,
  "zoom_factor": 1.5	// value from 0.5 -> 4
}
```​
*/
```

