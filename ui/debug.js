/*global console, Chart*/
function timeFmt(usValue) {
    "use strict";
    var unit = 'us';
    if (usValue > 1000) {
        usValue /= 1000.0;
        unit = 'ms';
    }
    if (usValue > 1000) {
        usValue /= 1000.0;
        unit = 's';
    }
    return usValue.toFixed(3) + unit;
}

function createHistogram(id, data, stats, noaxis) {
    "use strict";
    var barChartData = {
        labels: ['0-1us', '1-2us', '2-4us', '4-8us', '8-16us', '16-32us', '32-64us', '64-128us', '128-256us', '256-512us', '512us-1ms', '1-2ms', '2-4ms', '4-8ms', '8-16ms', '16-32ms', '32-64ms', '64-128ms', '128-256ms', '256-512ms', '512ms-1s', '1-2s'],
        datasets: [{
            label: 'Operations',
            backgroundColor: 'blue',
            borderColor: 'blue',
            borderWidth: 1,
            data: data
        }]
    };
    var options = {
        responsive: true,
        legend: {
            display: false
        },
        title: {
            display: false
        },
        scales: {
            xAxes: [{gridLines: {display: false}}],
            yAxes: [{gridLines: {display: false}}]
        }
    };
    if (noaxis) {
        options.scales.xAxes[0].display = false;
        options.scales.yAxes[0].display = false;
    }
    var ctx = document.getElementById(id).getContext('2d');
    document.getElementById(id + '-min').innerText = timeFmt(stats.min);
    document.getElementById(id + '-avg').innerText = timeFmt(stats.average);
    document.getElementById(id + '-max').innerText = timeFmt(stats.max);

    return new Chart(ctx, {
        type: 'bar',
        data: barChartData,
        options: options
    });
}


function createPieChart(id, series) {
    "use strict";

    var colors = ['red', 'green', 'blue', 'orange', 'purple', 'yellow'];

    var chartData = {
        labels: [],
        datasets: [{data: [], backgroundColor: []}]
    };

    series.forEach(function (item, colorIndex) {
        chartData.labels.push(item.name);
        chartData.datasets[0].data.push(item.value);
        chartData.datasets[0].backgroundColor.push(colors[colorIndex % colors.length]);
    });
    var options = {
        responsive: true,
        legend: {
            display: false
        },
        title: {
            display: false
        }
    };
    var ctx = document.getElementById(id).getContext('2d');

    return new Chart(ctx, {
        type: 'pie',
        data: chartData,
        options: options
    });
}
// Create created/updated/deleted charts for apps, gateways, devices.
function createStackedBars(id, series) {
    "use strict";
    var colors = ['red', 'green', 'blue', 'orange', 'purple', 'yellow'];
    var legend = [];
    var i = 60;
    while (i > 0) {
        i = i - 1;
        legend.push(i + '');
    }
    legend.push('now');

    var barChartData = {
        labels: legend,
        datasets: []
    };

    series.forEach(function (val, colorIndex) {
        barChartData.datasets.push({
            label: val.name,
            backgroundColor: colors[colorIndex % colors.length],
            borderColor: colors[colorIndex % colors.length],
            borderWidth: 1,
            data: val.data
        });
    });

    var options = {
        responsive: true,
        legend: {
            display: true,
            position: 'right'
        },
        title: {
            display: false
        },
        scales: {
            xAxes: [{stacked: true, gridLines: {display: false}}],
            yAxes: [{stacked: true}]
        }
    };
    var ctx = document.getElementById(id).getContext('2d');
    return new Chart(ctx, {
        type: 'bar',
        data: barChartData,
        options: options
    });
}

function loadData(callbackOnComplete) {
    "use strict";
    var req = new XMLHttpRequest();
    req.open('GET', 'http://localhost:8081/debug/vars');
    req.setRequestHeader("Content-Type", "application/json");
    req.onload = function () {
        if (req.status === 200) {
            callbackOnComplete(JSON.parse(req.response));
        } else {
            console.log('Expected 200 from server but got ' + req.status);
        }
    };
    req.onerror = function () {
        console.log('Bummer (got error when doing request)');
    };
    req.send();
}

