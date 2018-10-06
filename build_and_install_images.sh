

#for t in "model" "objective" 
#do
#    for d in `cat ./modules/${t}s/list`
#    do
#        echo "Install module [" $t "]: " $d
#        cd ./modules/${t}s/$d
#        docker build -t $d .
#        cd ..
#        easeml create module --type $t --source upload --source-address $d --id $d --name $d
#        cd ../..
#    done
#done


for t in "optimizer"
do
    for d in `cat ./modules/${t}s/list`
    do
	echo "Install module [" $t "]: " $d
	cd ./modules/${t}s/$d
	docker build -t $d .
	cd ..
	easeml create module --type $t --source upload --source-address $d --id opt-$d --name opt-$d
	cd ../..
    done
done



for t in "objective" "model"  
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



