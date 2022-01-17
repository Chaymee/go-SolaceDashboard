
document.addEventListener('astilectron-ready', function() {
    // listen on message from GO
    astilectron.onMessage(function(message) {
        //process the message
        console.log(message);
        //loadTableData(message);
        mulipleLoad(message);
        //var table = document.getElementById('table').getElementsByTagName('tbody')[0];
        //var row = table.insertRow();
        //let contents = row.insertCell(0);
        //contents.innerHTML = message;
        //return
    })
})

function loadTableData(items) {
    const table = document.getElementById("firstTablebody");
    let count = table.rows.length;
    var tableHeaderRowCount = 1;
    console.log("Row count: " + count);
    if (count > 5) {
        for (var i = tableHeaderRowCount; i < count; i++) {
            table.deleteRow(tableHeaderRowCount);
        }
    }
    
    let row = table.insertRow();
    let button = row.insertCell(0);
    button.innerHTML = '<button onclick="removeRow('+ items[0] + ')">Delivered</button>';
    let number = row.insertCell(1);
    number.innerHTML = items[0];
    let contents = row.insertCell(2);
    contents.innerHTML = items[1];
}

function mulipleLoad(items) {
    const table = document.getElementById("firstTablebody");
    let count = table.rows.length;
    var tableHeaderRowCount = 1;
    console.log("Clearing table");
    for (var i = tableHeaderRowCount; i < count; i++) {
        table.deleteRow(tableHeaderRowCount);
    }

    console.log("Building table");
    for (let i= 0; i < items.length; i++) {
        if (items[i][0] != '') {
            let row = table.insertRow();
            let button = row.insertCell(0);
            button.innerHTML = '<button onclick="removeRow('+ items[i][0] + ')">Delivered</button>';
            let number = row.insertCell(1);
            number.innerHTML = items[i][0];
            let contents = row.insertCell(2);
            contents.innerHTML = items[i][1];
        }

    }
}



function removeRow(selectedRow) {
    // This will wait for the astilectron namespace to be ready
    //document.addEventListener('astilectron-ready', function(removeRow) {
        astilectron.sendMessage(selectedRow, function(message) {
            console.log("Selected row: " + selectedRow);
        });
    //});
}
