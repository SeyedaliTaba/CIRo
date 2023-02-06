#!/bin/bash

KEY="I8zTCMORZD5xSrV4HeasTnUqYoRIhgsb"

for d in $(ls ./gen/); do
    if [[ $d == *"AS"* ]];
    then
        printf "{\n    \"key\": \"$KEY\"\n}" > "./gen/$d/electricityMapsAPIKeyFile.json"
    fi
done
