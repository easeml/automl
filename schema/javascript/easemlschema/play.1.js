'use strict';

var opener = require('./fs-opener');


var r = opener(__dirname, "bla.txt")

var l = r.readLines();

console.log(l);
