const { createClient } = require('redis');
const LIST = "demo"


module.exports = {
  main: async function (event, _) {

    const port = process.env["REDIS_PORT"];
    const host = process.env["REDIS_HOST"];
    const password = process.env["REDIS_PASSWORD"];

    const client = createClient({
      password,
      socket: {
          host,
          port,
      }
    });

    client.on('error', err => console.log('Redis Client Error', err));

    await client.connect();

    var msg=event.extensions.request.body.msg
    console.log(msg); 
    const res1 = await client.lPush(LIST, msg);
    console.log(res1); 

  }
}

