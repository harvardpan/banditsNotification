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

function logMessage(message) {
  const timestamp = moment().tz(config.display_time_zone).format('dddd, MMMM Do YYYY, h:mm:ss a');
  console.log(`INFO: ${timestamp} - ${message}`);
}

async function tweetScreenshot(imageBuffer) {
  const client = new TwitterApi({
    appKey: config.consumer_key,
    appSecret: config.consumer_secret,
    accessToken: config.access_token_key,
    accessSecret: config.access_token_secret,
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
    text: `Latest Bandits 12U Schedule as of ${timestamp}. https://www.brooklinebaseball.net/bandits12u #bandits12u`,
    media: {media_ids: mediaIds},
  });

  logMessage(`Your image tweet has successfully posted`);
}

async function main() {
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  try {
    const page = await browser.newPage();
    await page.goto('https://www.brooklinebaseball.net/bandits12u');
    // Grab the page's HTML data
    const pageData = await page.evaluate(() => {
      return {html: document.documentElement.innerHTML};
    });
    // Parse the data with "cheerio" library
    const $ = cheerio.load(pageData.html);
    const scheduleNode = $('h5:contains("Upcoming Schedule")').parent(); // contains the entire schedule section
    const schedule = parseSchedule(scheduleNode.text());
    const scheduleDiff = await diffSchedule(schedule);
    if (!scheduleDiff.added.size && !scheduleDiff.deleted.size && !scheduleDiff.modified.size) {
      // If there are no changes, then we don't need to do anything.
      logMessage(`No differences detected.`);
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
    await uploadFileToS3(imageBuffer, `${config.twitterUserHandle}/archive/${screenshotFilenameBase}`);
    await serializeSchedule(schedule, `${config.twitterUserHandle}/previousSchedule.json`);
    await serializeSchedule(schedule, `${config.twitterUserHandle}/archive/${scheduleFilenameBase}`);
    await tweetScreenshot(imageBuffer);
  } catch (e) {
    logMessage('ERROR: Uncaught exception occurred');
    console.log(e);
  } finally {
    await browser.close();
  }
}

function sleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

(async () => {
  await init(); // connect to HCP Vault Secrets and populate environment variables

  while (true) {
    await main();
    await sleep(config.runInterval * 1000); // multiply by 1000 as sleep takes milliseconds
  }
})();
