#!/bin/sh

for i in `cat links.txt`;do 
    URL=$(echo "{\"url\":\"${i}\"}");
    curl https://linkfn.makerdev.nl/ -d $URL;
done