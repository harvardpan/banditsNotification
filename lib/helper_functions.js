/* eslint-disable require-jsdoc */
/* eslint-disable max-len */
const fs = require('fs');
const path = require('path');
const moment = require('moment-timezone');
const chrono = require('chrono-node');
const {EJSON} = require('bson');

const PREVIOUS_SCHEDULE_FILENAME = path.join(__dirname, '../', 'previousSchedule.json'); // "../" because this is the lib folder

function sanitizeText(text) {
  if (!text) {
    return text;
  }
  return text.replace(/[\u200B-\u200D\uFEFF]/g, '').replace(/–/, '-').trim();
}

function parseSchedule(text) {
  // Schedule starts with "Upcoming Schedule" and is bookended by "Schedule by Season"
  const results = text.split(/(Schedule by Season)/);
  const upcomingSchedule = results[0];
  const entries = upcomingSchedule.split(/((SUNDAY|MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY), +(\d+\/\d+))/).slice(1);
  const schedule = new Map(); // map of days to schedule information
  for (let i = 0; i < entries.length; i += 4) {
    const timeBlockMatch = entries[i + 3].match(/\d+:\d+([-–]\d+:\d+)?/);
    let timeBlock = null;
    if (timeBlockMatch) {
      timeBlock = timeBlockMatch[0];
    }
    const dayOfWeek = entries[i + 1];
    const dayOfMonth = entries[i + 2];
    let location = sanitizeText(entries[i + 3]); // defaults to the entire block of information
    if (timeBlock) {
      // A timeblock exists, so location is before it.
      location = entries[i + 3].split(timeBlock)[0].trim().replace(/, *$/, '');
    }
    const parsed = timeBlock ? chrono.parse(`${dayOfMonth} ${timeBlock}pm`) : null;
    const obj = {
      dayOfWeek,
      dayOfMonth,
      location,
      timeBlock,
      parsed,
    };
    schedule.set(`${entries[i]}`, obj);
  }
  return schedule;
}

function compareSchedules(a, b) {
  // eslint-disable-next-line one-var, prefer-const
  let added = new Map(), deleted = new Map(), modified = new Map(), unchanged = new Map();
  if (!a) {
    // When "a" isn't valid, we just add everything into "added"
    added = new Map(b.entries());
  } else {
    a.forEach(function(value, key, map) {
      if (!b.has(key)) {
        deleted.set(key, value);
      }
    });
    b.forEach(function(value, key, map) {
      if (!a.has(key)) {
        added.set(key, value);
        return;
      }
      // If the key already exist, check if it was modified or unchanged.
      const aValue = a.get(key);
      if (aValue['location'] !== value['location'] || aValue['timeBlock'] !== value['timeBlock']) {
        modified.set(key, value);
      } else {
        unchanged.set(key, value);
      }
    });
  }
  return {added, deleted, modified, unchanged};
}

function serializeSchedule(schedule, filepath) {
  const data = EJSON.stringify(schedule);
  fs.writeFileSync(filepath, data);
  return data;
}

function deserializeSchedule(filepath) {
  const data = fs.readFileSync(filepath, 'utf-8');
  const scheduleObject = EJSON.parse(data);

  // Convert the Object => Map
  const schedule = new Map();
  for (const key in scheduleObject) {
    if (!Object.prototype.hasOwnProperty.call(scheduleObject, key)) {
      continue;
    }
    schedule.set(key, scheduleObject[key]);
  }
  return schedule;
}

function getTimestampedFilename(filenameBase = 'schedule-screenshot', extension = 'png') {
  const timestamp = Date.now();

  const dateObject = new Date(timestamp);
  const date = dateObject.getDate();
  const month = dateObject.getMonth() + 1;
  const year = dateObject.getFullYear();

  // prints date & time in YYYY-MM-DD format, plus the milliseconds to differentiate
  return `${filenameBase}-${year}-${month}-${date}-${moment().valueOf()}.${extension}`;
}

/**
 * Compares the passed schedule with the prior schedule
 *
 * @param {Map} schedule the schedule Map object that should be compared
 * @return {Object} the output of comparing the schedule with the previous schedule
 */
function diffSchedule(schedule) {
  if (!fs.existsSync(PREVIOUS_SCHEDULE_FILENAME)) {
    // Usually, if the previous schedule doesn't exist, this is the first
    // time that this is running in the docker container. Will not need this
    // logic anymore once we move the previous blobs to S3
    serializeSchedule(schedule, PREVIOUS_SCHEDULE_FILENAME);
  }
  const previousSchedule = deserializeSchedule(PREVIOUS_SCHEDULE_FILENAME);
  return compareSchedules(previousSchedule, schedule);
}

module.exports = {
  parseSchedule,
  compareSchedules,
  serializeSchedule,
  deserializeSchedule,
  getTimestampedFilename,
  diffSchedule,
};
