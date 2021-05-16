import json
import requests
import sys

statesURL = "https://cdn-api.co-vin.in/api/v2/admin/location/states"
districtURL = "https://cdn-api.co-vin.in/api/v2/admin/location/districts/"
headers = {"User-Agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36"}
stateResult = requests.get(url=statesURL, headers=headers)
stateData = stateResult.json()

file = open("/Users/kumarsaras/projects/covaxinate/utils/jsontemplates/districttemplate.json")
data = json.load(file)
newOptions = data['elements'][0]['options']


districtURL = districtURL + str(sys.argv[1])
districtResult = requests.get(url=districtURL, headers=headers)
districtData = districtResult.json()
#print(districtData)
for district in districtData['districts']:
    distName = district['district_name']
    distID = str(district['district_id'])
    option = {"label": district['district_name']+"("+distID+")", "value": distID}
    newOptions.append(option)
data['elements'][0]['options'] = newOptions
districtsFileName = "/Users/kumarsaras/projects/covaxinate/utils/jsontemplates/" + sys.argv[1] + ".json"
with open(districtsFileName, "w") as outfile: 
    json.dump(data, outfile)

