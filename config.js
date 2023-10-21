/* eslint-disable max-len */
require('dotenv').config();


/**
 * Javascript Class file for Config object. Uses getters to have a
 * dynamic behavior around the configs, so that it can be re-evaluated
 * every time instead of being a static value. Supports the potential
 * use of feature flags in the future.
 *
 * @class Config
 * @typedef {Config}
 */
class Config {
  /**
   * Creates an instance of Config.
   *
   * @constructor
   */
  constructor() {
  }

  /**
   * Retrieves the Twitter Consumer Key - API Key
   *
   * @readonly
   * @type {String}
   */
  get consumer_key() {
    return process.env.TWITTER_CONSUMER_KEY;
  }

  /**
   * Retrieves the Twitter Consumer Secret - API Key Secret
   *
   * @readonly
   * @type {String}
   */
  get consumer_secret() {
    return process.env.TWITTER_CONSUMER_SECRET;
  }

  /**
   * Retrieves the Twitter Access Token Key
   *
   * @readonly
   * @type {String}
   */
  get access_token_key() {
    return process.env.TWITTER_ACCESS_TOKEN_KEY;
  }

  /**
   * Retrieves the Twitter Access Token Secret
   *
   * @readonly
   * @type {String}
   */
  get access_token_secret() {
    return process.env.TWITTER_ACCESS_TOKEN_SECRET;
  }

  /**
   * Retrieves the # of seconds between checks/runs
   *
   * @readonly
   * @type {Integer}
   */
  get runInterval() {
    let interval = parseInt(process.env.RUN_INTERVAL);
    if (isNaN(interval)) {
      interval = 300; // default to 300 seconds when an invalid number is presented
    }
    return interval;
  }

  /**
   * Retrieves the Twitter User Handle that the posts should come from (i.e. name of
   * the bot). This is used primarily for testing connectivity in the tests.
   *
   * @readonly
   * @type {String}
   */
  get twitterUserHandle() {
    return process.env.TWITTER_USER_HANDLE;
  }

  /**
   * Retrieves the AWS Access Key ID. These are the same environment variables
   * that AWS SDK uses.
   *
   * @readonly
   * @type {String}
   */
  get aws_access_token_id() {
    return process.env.AWS_ACCESS_KEY_ID;
  }

  /**
   * Retrieves the AWS Secret Access Key. These are the same environment variables
   * that the AWS SDK uses.
   *
   * @readonly
   * @type {String}
   */
  get aws_access_token_secret() {
    return process.env.AWS_SECRET_ACCESS_KEY;
  }

  /**
   * Retrieves the AWS Default Region. These are the same environment variables
   * that the AWS SDK uses.
   *
   * @readonly
   * @type {String}
   */
  get aws_default_region() {
    let region = 'us-east-1'; // this is the default
    if (process.env.AWS_DEFAULT_REGION) {
      region = process.env.AWS_DEFAULT_REGION;
    }
    return region;
  }

  /**
   * Retrieves the AWS S3 Bucket name. This is where the archived screenshots
   * and json get uploaded.
   *
   * @readonly
   * @type {String}
   */
  get aws_s3_bucket() {
    return process.env.AWS_S3_BUCKET;
  }

  /**
   * Retrieves the HCP Client ID.
   *
   * @readonly
   * @type {String}
   */
  get hcp_client_id() {
    return process.env.HCP_CLIENT_ID;
  }

  /**
   * Retrieves the HCP Client Secret.
   *
   * @readonly
   * @type {String}
   */
  get hcp_client_secret() {
    return process.env.HCP_CLIENT_SECRET;
  }

  /**
   * Retrieves the HCP Organization Id
   *
   * @readonly
   * @type {String}
   */
  get hcp_organization_id() {
    return process.env.HCP_ORGANIZATION_ID;
  }

  /**
   * Retrieves the HCP Project Id
   *
   * @readonly
   * @type {String}
   */
  get hcp_project_id() {
    return process.env.HCP_PROJECT_ID;
  }

  /**
   * Retrieves the HCP Application Name
   *
   * @readonly
   * @type {String}
   */
  get hcp_application_name() {
    return process.env.HCP_APP_NAME;
  }
}

module.exports = new Config();
