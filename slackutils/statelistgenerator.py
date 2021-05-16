import json
import requests

statesURL = "https://cdn-api.co-vin.in/api/v2/admin/location/states"
headers = {"User-Agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36"}
stateResult = requests.get(url=statesURL, headers=headers)
stateData = stateResult.json()

file = open("/Users/kumarsaras/projects/covaxinate/utils/jsontemplates/statetemplate.json")
data = json.load(file)
newOptions = data['blocks'][1]['accessory']['options']

for state in stateData['states']:
    stateName = state['state_name']
    stateID = str(state['state_id'])
    option = {"text":{"type": "plain_text", "text": state['state_name']+"("+stateID+")", "emoji":True}, "value": stateID}
    newOptions.append(option)
data['blocks'][1]['accessory']['options'] = newOptions

with open("/Users/kumarsaras/projects/covaxinate/utils/jsontemplates/allstates.json", "w") as outfile: 
    json.dump(data, outfile, indent=4)

