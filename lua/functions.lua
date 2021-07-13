json = require './lua/json'

function sum(incomingData) 
    dataTable = json.decode(incomingData)
    return tostring(tonumber(dataTable[1]) + tonumber(dataTable[2]))
end

function mul(incomingData) 
    dataTable = json.decode(incomingData)
    return tostring(tonumber(dataTable[1]) * tonumber(dataTable[2]))
end