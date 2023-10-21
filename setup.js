/* eslint-disable max-len */
const {retrieveApiToken, retrieveSecret} = require('./lib/vault');

/**
 * Makes a call to HCP Vault Secrets and populates the environment variables
 *
 * @async
 */
async function init() {
  const apiToken = await retrieveApiToken();
  process.env.TWITTER_CONSUMER_KEY = await retrieveSecret(apiToken, 'TWITTER_CONSUMER_KEY');
  process.env.TWITTER_CONSUMER_SECRET = await retrieveSecret(apiToken, 'TWITTER_CONSUMER_SECRET');
  process.env.TWITTER_ACCESS_TOKEN_KEY = await retrieveSecret(apiToken, 'TWITTER_ACCESS_TOKEN_KEY');
  process.env.TWITTER_ACCESS_TOKEN_SECRET = await retrieveSecret(apiToken, 'TWITTER_ACCESS_TOKEN_SECRET');
  process.env.TWITTER_USER_HANDLE = await retrieveSecret(apiToken, 'TWITTER_USER_HANDLE');
  process.env.AWS_ACCESS_KEY_ID = await retrieveSecret(apiToken, 'AWS_ACCESS_KEY_ID');
  process.env.AWS_SECRET_ACCESS_KEY = await retrieveSecret(apiToken, 'AWS_SECRET_ACCESS_KEY');
  process.env.AWS_DEFAULT_REGION = await retrieveSecret(apiToken, 'AWS_DEFAULT_REGION');
  process.env.AWS_S3_BUCKET = await retrieveSecret(apiToken, 'AWS_S3_BUCKET');
}

module.exports = {
  init,
};
