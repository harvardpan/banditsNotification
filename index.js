/* eslint-disable max-len */
/* eslint-disable require-jsdoc */
'use strict';
const puppeteer = require('puppeteer');
const {TwitterApi} = require('twitter-api-v2');
const config = require('./config');
const moment = require('moment-timezone');
const cheerio = require('cheerio');
const {
  parseSchedule,
  getTimestampedFilename,
  diffSchedule,
  serializeSchedule,
} = require('./lib/helper_functions');
const {
  uploadFileToS3,
} = require('./lib/aws');
const {init} = require('./setup');
const {retrieveApiToken, retrieveSecret} = require('./lib/vault');

function logMessage(message) {
  const timestamp = moment().tz(config.display_time_zone).format('dddd, MMMM Do YYYY, h:mm:ss a');
  console.log(`INFO: ${timestamp} - ${message}`);
}

async function tweetScreenshot(imageBuffer, settings) {
  const twitterConsumerKey = await retrieveSecret(settings.apiToken, 'TWITTER_CONSUMER_KEY', settings.hcp_app_name);
  const twitterConsumerSecret = await retrieveSecret(settings.apiToken, 'TWITTER_CONSUMER_SECRET', settings.hcp_app_name);
  const twitterAccessTokenKey = await retrieveSecret(settings.apiToken, 'TWITTER_ACCESS_TOKEN_KEY', settings.hcp_app_name);
  const twitterAccessTokenSecret = await retrieveSecret(settings.apiToken, 'TWITTER_ACCESS_TOKEN_SECRET', settings.hcp_app_name);

  const client = new TwitterApi({
    appKey: twitterConsumerKey,
    appSecret: twitterConsumerSecret,
    accessToken: twitterAccessTokenKey,
    accessSecret: twitterAccessTokenSecret,
  });

  // First, post all your images to Twitter
  const mediaIds = await Promise.all([
    // file path
    client.v1.uploadMedia(Buffer.from(imageBuffer), {
      type: 'png',
    }),
  ]);

  const timestamp = moment().tz(config.display_time_zone).format('dddd, MMMM Do YYYY, h:mm:ss a');
  // mediaIds is a string[], can be given to .tweet
  await client.v2.tweet({
    text: `Latest Bandits Schedule as of ${timestamp}. ${settings.url}`,
    media: {media_ids: mediaIds},
  });

  logMessage(`Your image tweet has successfully posted`);
}

/**
 * 
 * @param {Object} settings A settings objects which contains the following properties:
 * - url: The URL of the Bandits site to check
 * - hcp_app_name: The name of the HCP app to use for Twitter handle and secrets
 * - apiToken: The API token to use for HCP Vault Secrets, passed from the global scope
 * @returns 
 */
async function checkBanditsSite(settings) {
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  try {
    const page = await browser.newPage();
    await page.goto(settings.url); // Go to the specified Bandits site based on URL
    // Grab the page's HTML data
    const pageData = await page.evaluate(() => {
      return {html: document.documentElement.innerHTML};
    });
    // Parse the data with "cheerio" library
    const $ = cheerio.load(pageData.html);
    const scheduleNode = $('h5:contains("Upcoming Schedule")').parent(); // contains the entire schedule section
    const schedule = parseSchedule(scheduleNode.text());
    // Retrieve the Twitter secrets from HCP Vault Secrets
    const twitterUserHandle = await retrieveSecret(settings.apiToken, 'TWITTER_USER_HANDLE', settings.hcp_app_name);
    const scheduleDiff = await diffSchedule(schedule, twitterUserHandle);
    if (!scheduleDiff.added.size && !scheduleDiff.deleted.size && !scheduleDiff.modified.size) {
      // If there are no changes, then we don't need to do anything.
      logMessage(`No differences detected. No need to publish notification.`);
      return;
    }

    // Below here, a difference was detected, so we take a screenshot.

    // Grab only the screen part relevant to the schedule
    await page.setViewport({width: 1200, height: 800, deviceScaleFactor: 2});

    const screenshotFilenameBase = getTimestampedFilename('schedule-screenshot', 'png');
    const scheduleFilenameBase = screenshotFilenameBase.replace(/.png$/, '.json').replace(/-screenshot/, '');
    // Take the screenshot of the portion of the screen with the schedule
    const imageBuffer = await page.screenshot({
      type: 'png',
      //      path: screenshotFilename,
      clip: {
        height: 470,
        width: 340,
        x: 150,
        y: 200,
      },
      omitBackground: true,
    });
  
    // Since a diff was detected, we want to:
    // - upload the latest screenshot to the archive
    // - serialize the schedule json
    // - copy the schedule json to the archive
    // - tweet out the latest screenshot
    await uploadFileToS3(imageBuffer, `${twitterUserHandle}/archive/${screenshotFilenameBase}`);
    await serializeSchedule(schedule, `${twitterUserHandle}/previousSchedule.json`);
    await serializeSchedule(schedule, `${twitterUserHandle}/archive/${scheduleFilenameBase}`);
    await tweetScreenshot(imageBuffer, settings);
  } catch (e) {
    logMessage('ERROR: Uncaught exception occurred');
    console.log(e);
  } finally {
    await browser.close();
  }
}
async function main(apiToken) {
  await checkBanditsSite({ 
    url: 'https://www.brooklinebaseball.net/bandits12u',
    hcp_app_name: 'BanditsNotificationBot',
    apiToken: apiToken
  });
  await checkBanditsSite({ 
    url: 'https://www.brooklinebaseball.net/bandits14u',
    hcp_app_name: 'BanditsNotification14UBot',
    apiToken: apiToken
  });
}

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

(async () => {
  await init(); // connect to HCP Vault Secrets and populate environment variables
  const apiToken = await retrieveApiToken();

  while (true) {
    await main(apiToken);
    await sleep(config.runInterval * 1000); // multiply by 1000 as sleep takes milliseconds
  }
})();
