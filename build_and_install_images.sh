

for t in "model" "objective" "optimizer"
do
    for d in `cat ./modules/${t}s/list`
    do
	echo "Install module [" $t "]: " $d
	cd ./modules/${t}s/$d
	docker build -t $d .
	cd ..
	easeml create module --type $t --source upload --source-address $d --id $d --name $d
	cd ../..
    done
done

