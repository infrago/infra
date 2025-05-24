#!/usr/bin/env bash
v="v0.2.0"

cd ..

for file in `ls`
do
    if [ -d $file ]
    then
        cd $file
        git push origin $v
        echo $file $v
        cd ..
    fi
done

# echo $v
