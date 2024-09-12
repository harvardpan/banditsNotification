/* eslint-disable max-len */
const {retrieveApiToken, retrieveSecret} = require('./lib/vault');

/**
 * Makes a call to HCP Vault Secrets and populates the environment variables
 *
 * @async
 */
async function init() {
  const apiToken = await retrieveApiToken();
  // The AWS S3 environment variables are retrieved from the HCP Vault "AWS-S3-Access" app
  process.env.AWS_ACCESS_KEY_ID = await retrieveSecret(apiToken, 'AWS_ACCESS_KEY_ID');
  process.env.AWS_SECRET_ACCESS_KEY = await retrieveSecret(apiToken, 'AWS_SECRET_ACCESS_KEY');
  process.env.AWS_DEFAULT_REGION = await retrieveSecret(apiToken, 'AWS_DEFAULT_REGION');
  process.env.AWS_S3_BUCKET = await retrieveSecret(apiToken, 'AWS_S3_BUCKET');
}

module.exports = {
  init,
};
