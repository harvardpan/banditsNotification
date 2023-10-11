const expect = require('chai').expect;
const {TwitterApi} = require('twitter-api-v2');
const config = require('../config');

describe('Twitter Integration Tests', function() {
  it(`can connect to Twitter`, async function() {
    // Construct the Client
    const client = new TwitterApi({
      appKey: config.consumer_key,
      appSecret: config.consumer_secret,
      accessToken: config.access_token_key,
      accessSecret: config.access_token_secret,
    });
    // Retrieve the account settings
    const settings = await client.v1.accountSettings();
    expect(settings.screen_name).to.equal(config.twitterUserHandle);
  });
});
