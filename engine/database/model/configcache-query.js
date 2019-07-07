DBQuery.shellBatchSize = 3000
use easeml;
db.tasks.aggregate(
    [
        {
            $match : {
                "model" : {$in : ["root/mdl-reg-gp"]}
            }
        },
        {
            $group : {
                _id : {"model" : "$model", "config" : "$config", "dataset" : "$dataset", "objective": "$objective"},
                "avg-quality": {$avg : "$quality"},
                "cum-quality" : {$sum : "$quality"},
                count : {$sum : 1}
            }
        },
        {
            $group : {
                _id : {"model" : "$_id.model", "config" : "$_id.config", "dataset" : "$_id.dataset"},
                "cum-quality" : {$sum : "$cum-quality"},
                count : {$sum : "$count"},
                quality : { $push : { "k" : "$_id.objective" , "v" : "$avg-quality" } }
            }
        },
        {
            $group : {
                _id : {"model" : "$_id.model", "config" : "$_id.config"},
                "cum-quality" : {$sum : "$cum-quality"},
                count : {$sum : "$count"},
                quality : { $push : { "k" : "$_id.dataset" , "v" : { $arrayToObject : "$quality" } } }
            }
        },
        {
            $project : {
                _id: 0,
                model : "$_id.model",
                config : "$_id.config",
                "avg-quality": {$divide : ["$cum-quality", "$count"]},
                count : 1,
                quality : { $arrayToObject : "$quality" }
            }
        }
    ]
)
