'use strict';

var opener = require('./fs-opener');
var jsnpy = require('./jsnpy');


var r = opener(__dirname, "td1.ten.npy", "tensor", true);
var nr = new jsnpy.NpyReader(r);
var data = nr.read();

var w = opener(__dirname, "deleteme.npy", "tensor", false);
var nw = new jsnpy.NpyWriter(w, nr.shape, nr.dtype, nr.column_major, nr.big_endian, nr.version);
nw.write(data);
