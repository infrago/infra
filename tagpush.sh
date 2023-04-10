#!/usr/bin/env bash
cd ..

for file in `ls`
do
    if [ -d $file ]
    then
        cd $file
        git push origin $1
        echo $file $1
        cd ..
    fi
done

# echo $1
