/**
 * Theme: Highdmin - Responsive Bootstrap 4 Admin Dashboard
 * Author: Coderthemes
 * Module/App: Flot-Chart
 */

! function($) {
    "use strict";

    var FlotChart = function() {
        this.$body = $("body")
        this.$realData = []
    };

    //creates plot graph
    FlotChart.prototype.createPlotGraph = function(selector, data1, data2, data3, labels, colors, borderColor, bgColor) {
        //shows tooltip
        function showTooltip(x, y, contents) {
            $('<div id="tooltip" class="tooltipflot">' + contents + '</div>').css({
                position : 'absolute',
                top : y + 5,
                left : x + 5
            }).appendTo("body").fadeIn(200);
        }


        $.plot($(selector), [{
            data : data1,
            label : labels[0],
            color : colors[0]
        }, {
            data : data2,
            label : labels[1],
            color : colors[1]
        },
            {
                data : data3,
                label : labels[2],
                color : colors[2]
            }], {
            series : {
                lines : {
                    show : true,
                    fill : true,
                    lineWidth : 2,
                    fillColor : {
                        colors : [{
                            opacity : 0.5
                        }, {
                            opacity : 0.5
                        }, {
                            opacity: 0.8
                        }]
                    }
                },
                points : {
                    show : true
                },
                shadowSize : 0
            },

            grid : {
                hoverable : true,
                clickable : true,
                borderColor : borderColor,
                tickColor : "#f9f9f9",
                borderWidth : 1,
                labelMargin : 30,
                backgroundColor : bgColor
            },
            legend : {
                position: "ne",
                margin : [0, -32],
                noColumns : 0,
                labelBoxBorderColor : null,
                labelFormatter : function(label, series) {
                    // just add some space to labes
                    return '' + label + '&nbsp;&nbsp;';
                },
                width : 30,
                height : 2
            },
            yaxis : {
                axisLabel: "Daily Visits",
                tickColor : '#f5f5f5',
                font : {
                    color : '#bdbdbd'
                }
            },
            xaxis : {
                axisLabel: "Last Days",
                tickColor : '#f5f5f5',
                font : {
                    color : '#bdbdbd'
                }
            },
            tooltip : true,
            tooltipOpts : {
                content : '%s: Value of %x is %y',
                shifts : {
                    x : -60,
                    y : 25
                },
                defaultTheme : false
            },
            splines: {
                show: true,
                tension: 0.1, // float between 0 and 1, defaults to 0.5
                lineWidth: 1 // number, defaults to 2
            }
        });
    },
        //end plot graph

        //creates Donut Chart
        FlotChart.prototype.createDonutGraph = function(selector, labels, datas, colors) {
            var data = [{
                label : labels[0],
                data : datas[0]
            }, {
                label : labels[1],
                data : datas[1]
            }, {
                label : labels[2],
                data : datas[2]
            },{
                label : labels[3],
                data : datas[3]
            }, {
                label : labels[4],
                data : datas[4]
            }];
            var options = {
                series : {
                    pie : {
                        show : true,
                        innerRadius : 0.7
                    }
                },
                legend : {
                    position: "sw",
                    margin : [0, 0],
                    noColumns : 2,
                    show : false,
                    labelFormatter : function(label, series) {
                        return '<div style="font-size:14px;">&nbsp;' + label + '</div>'
                    },
                    labelBoxBorderColor : null,
                    width : 20
                },
                grid : {
                    hoverable : true,
                    clickable : true
                },
                colors : colors,
                tooltip : true,
                tooltipOpts : {
                    content : "%s, %p.0%"
                }
            };

            $.plot($(selector), data, options);
        },
        //creates Combine Chart
        FlotChart.prototype.createCombineGraph = function(selector, ticks, labels, datas) {

            var data = [{
                label : labels[0],
                data : datas[0],
                lines : {
                    show : true,
                    fill : true
                },
                points : {
                    show : true
                }
            }, {
                label : labels[1],
                data : datas[1],
                lines : {
                    show : true
                },
                points : {
                    show : true
                }
            }, {
                label : labels[2],
                data : datas[2],
                bars : {
                    show : true
                }
            }];
            var options = {
                series : {
                    shadowSize : 0
                },
                grid : {
                    hoverable : true,
                    clickable : true,
                    tickColor : "#f9f9f9",
                    borderWidth : 1,
                    borderColor : "#eeeeee"
                },
                colors : ['#e3eaef','#f1556c','#02c0ce'],
                tooltip : true,
                tooltipOpts : {
                    defaultTheme : false
                },
                legend : {
                    position : "ne",
                    margin : [0, -32],
                    noColumns : 0,
                    labelBoxBorderColor : null,
                    labelFormatter : function(label, series) {
                        // just add some space to labes
                        return '' + label + '&nbsp;&nbsp;';
                    },
                    width : 30,
                    height : 2
                },
                yaxis : {
                    axisLabel: "Point Value (1000)",
                    tickColor : '#f5f5f5',
                    font : {
                        color : '#bdbdbd'
                    }
                },
                xaxis : {
                    axisLabel: "Daily Hours",
                    ticks: ticks,
                    tickColor : '#f5f5f5',
                    font : {
                        color : '#bdbdbd'
                    }
                }
            };

            $.plot($(selector), data, options);
        },

        //initializing various charts and components
        FlotChart.prototype.init = function() {
            //plot graph data
            var uploads = [[0, 13], [1, 13], [2, 14], [3, 62], [4, 13], [5, 10], [6, 56],[7, 13], [8, 12], [9, 20], [10, 48], [11, 16], [12, 14]];
            var downloads = [[0, 8], [1, 10], [2, 12], [3, 14], [4, 36], [5, 7], [6, 9],[7, 10], [8, 41], [9, 17], [10, 15], [11, 13], [12, 11]];
            var downloads1 = [[0, 3], [1, 22], [2, 8], [3, 10], [4, 7], [5, 3], [6, 5],[7, 7], [8, 6], [9, 14], [10, 35], [11, 10], [12, 8]];
            var plabels = ["Bitcoin", "Ethereum", "Litecoin"];
            var pcolors = ['#02c0ce','#2d7bf4','#f1556c'];
            var borderColor = '#f5f5f5';
            var bgColor = '#fff';
            this.createPlotGraph("#website-stats", uploads, downloads,downloads1, plabels, pcolors, borderColor, bgColor);

            //Combine graph data
            var data24Hours = [[0, 201], [1, 520], [2, 337], [3, 261], [4, 157], [5, 95], [6, 200], [7, 250], [8, 320], [9, 500], [10, 152], [11, 214], [12, 364], [13, 449], [14, 558], [15, 282], [16, 379], [17, 429], [18, 518], [19, 470], [20, 330], [21, 245], [22, 358], [23, 74]];
            var data48Hours = [[0, 311], [1, 630], [2, 447], [3, 371], [4, 267], [5, 205], [6, 310], [7, 360], [8, 430], [9, 610], [10, 262], [11, 324], [12, 474], [13, 559], [14, 668], [15, 392], [16, 489], [17, 539], [18, 628], [19, 580], [20, 440], [21, 355], [22, 468], [23, 184]];
            var dataDifference = [[23, 727], [22, 128], [21, 110], [20, 92], [19, 172], [18, 63], [17, 150], [16, 592], [15, 12], [14, 246], [13, 52], [12, 149], [11, 123], [10, 2], [9, 325], [8, 10], [7, 15], [6, 89], [5, 65], [4, 77], [3, 600], [2, 200], [1, 385], [0, 200]];
            var ticks = [[0, "22h"], [1, ""], [2, "00h"], [3, ""], [4, "02h"], [5, ""], [6, "04h"], [7, ""], [8, "06h"], [9, ""], [10, "08h"], [11, ""], [12, "10h"], [13, ""], [14, "12h"], [15, ""], [16, "14h"], [17, ""], [18, "16h"], [19, ""], [20, "18h"], [21, ""], [22, "20h"], [23, ""]];
            var combinelabels = ["Last 24 Hours", "Last 48 Hours", "Difference"];
            var combinedatas = [data24Hours, data48Hours, dataDifference];

            this.createCombineGraph("#combine-chart #combine-chart-container", ticks, combinelabels, combinedatas);


            //Donut pie graph data
            var donutlabels = ["Bitcoin", "Ethereum", "Litecoin", "Bitcoin Cash", "Cardano"];
            var donutdatas = [48, 30, 15, 32, 26];
            var donutcolors = ['#02c0ce','#2d7bf4','#e3eaef','#f1556c',"#f9bc0b"];
            this.createDonutGraph("#donut-chart #donut-chart-container", donutlabels, donutdatas, donutcolors);

        },

        //init flotchart
        $.FlotChart = new FlotChart, $.FlotChart.Constructor =
        FlotChart

}(window.jQuery),

//initializing flotchart
    function($) {
        "use strict";
        $.FlotChart.init()
    }(window.jQuery);