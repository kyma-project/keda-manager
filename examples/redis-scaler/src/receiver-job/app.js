const { createClient } = require('redis');
const port = process.env["REDIS_PORT"];
const host = process.env["REDIS_HOST"];
const password = process.env["REDIS_PASSWORD"];
const listName = "demo"
var msg = undefined;

async function popRedisList(host, port, password, listName) {

    const client = createClient({
        password,
        socket: {
            host,
            port,
        }
      });

    client.on('error', err => console.log('Redis Client Error', err));

    await client.connect();

    msg = await client.lPop(listName);

    let waitTime = between(1000, 10000); 
    console.log(`Processing started for ${msg}.. will finish in ${waitTime}ms`);

    await setTimeout(() => {}, waitTime)
    
    return;
}

popRedisList(host, port, password, listName).then(()=>{
    console.log(`Processing finished for ${msg}`);
    process.exit(0); 
})









function between(min, max) {
    return Math.floor(Math.random() * (max - min) + min);
}




