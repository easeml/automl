

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

config_file=$HOME/.easeml/config.yaml
#config_file=$HOME/snap/easeml/x1/.easeml/config.yaml

for t in "optimizer"
do
    for d in `cat ./modules/${t}s/list`
    do
	echo "Install module [" $t "]: " $d
	cd ./modules/${t}s/$d
	docker build -t $d .
	cd ..
	cmd="easeml create module --type $t --source upload --source-address $d --id opt-$d --label opt-$d --name opt-$d --config $config_file" 
	echo $cmd
	eval $cmd
	#read -p "Press enter to continue"
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
        cmd="easeml create module --type $t --source upload --source-address $d --id $d --label label-$d --name $d --config $config_file"
        echo $cmd
        eval $cmd
        #read -p "Press enter to continue"
        cd ../..
    done
done



