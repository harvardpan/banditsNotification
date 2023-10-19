/* eslint-disable max-len */
const expect = require('chai').expect;
const unroll = require('unroll');
unroll.use(it);
const moment = require('moment-timezone');
const fs = require('fs');
const path = require('path');
const {parseSchedule, compareSchedules, serializeSchedule, deserializeSchedule} = require('../lib/helper_functions');

describe('Helper Functions Unit Tests', function() {
  const input = [
    'Upcoming Schedule\n\nWear baseball pants or sweatpants to every practice, and bring all of your baseball gear.\n\n​\n\nTUESDAY, 10/3\n\nPractice, Warren, 4:45–6:45\n\nLate: Aiden, Sam, Zach\n\nOut: Matty\n\n​\n\nTHURSDAY, 10/5\n\nPractice, Warren, 4:45–6:45\n\nLate: —\n\nOut: Matty\n\n​\n\nSATURDAY, 10/7\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n​\n\nSUNDAY, 10/8\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n  \n\nSchedule by Season\n\nOur tentative plan for the months ahead. More details to come. \n\n \n\nFALL 20…Th/Sa/Su), September–November, at Warren Field (starting 9/5).\n\n \n\nWINTER 2024\n\nIndoor practices on Saturday or Sunday evenings, January–March, at Brookline HS Tappan Pavilion.\n\n \n\nSPRING 2024\n\n​Doubleheaders on Saturdays, April–June, in the 12U Division of the Select League.\n\n​\n\n\n\nPractices once or twice per week (TBD).\n\n\nPlayoffs in July, if we qualify.\n\n\nSchedule will not conflict with BYB Majors.\n\n\nPlaying time will depend on baseball skills, commitment, focus, work ethic, and attitude.​\n\n',
    'Upcoming Schedule\n\nWear baseball pants or sweatpants to every practice, and bring all of your baseball gear.\n\n​\n\nTHURSDAY, 10/5\n\nPractice, Warren, 4:30–6:30\n\nLate: —\n\nOut: Alek, Matty\n\n​\n\nSATURDAY, 10/7\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n​\n\nSUNDAY, 10/8\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n​\n\nTUESDAY, 10/10\n\nPractice, Warren, 4:30–6:30\n\nLate: Aiden, Sam, Zach\n\nOut: Matty\n\n​\n\nTUESDAY, 10/12\n\nPractice, Warren, 4:30–6:30\n\nLate: —\n\nOut: Matty\n\n  \n\nSchedule by Season\n…Th/Sa/Su), September–November, at Warren Field (starting 9/5).\n\n \n\nWINTER 2024\n\nIndoor practices on Saturday or Sunday evenings, January–March, at Brookline HS Tappan Pavilion.\n\n \n\nSPRING 2024\n\n​Doubleheaders on Saturdays, April–June, in the 12U Division of the Select League.\n\n​\n\n\n\nPractices once or twice per week (TBD).\n\n\nPlayoffs in July, if we qualify.\n\n\nSchedule will not conflict with BYB Majors.\n\n\nPlaying time will depend on baseball skills, commitment, focus, work ethic, and attitude.​\n\n',
    'Upcoming Schedule\n\nWear baseball pants or sweatpants to every practice, and bring all of your baseball gear.\n\n​\n\nSATURDAY, 10/7\n\nPractice is canceled\n\n​\n\nSUNDAY, 10/8\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor, Zach\n\n​\n\nTUESDAY, 10/10\n\nPractice, Warren, 4:30–6:30\n\nLate: Aiden, Ethan, Sam, Zach\n\nOut: Matty\n\n​\n\nTHURSDAY, 10/12\n\nPractice, Warren, 4:30–6:30\n\nLate: —\n\nOut: Matty\n\n  \n\nSchedule by Season\n\nOur tentative plan for the months ahead. More details to come. \n\n \n\nFALL 2023\n\nOutdoor pr…Th/Sa/Su), September–November, at Warren Field (starting 9/5).\n\n \n\nWINTER 2024\n\nIndoor practices on Saturday or Sunday evenings, January–March, at Brookline HS Tappan Pavilion.\n\n \n\nSPRING 2024\n\n​Doubleheaders on Saturdays, April–June, in the 12U Division of the Select League.\n\n​\n\n\n\nPractices once or twice per week (TBD).\n\n\nPlayoffs in July, if we qualify.\n\n\nSchedule will not conflict with BYB Majors.\n\n\nPlaying time will depend on baseball skills, commitment, focus, work ethic, and attitude.​\n\n',
    'Upcoming Schedule\n\nWear baseball pants or sweatpants to every practice, and bring all of your baseball gear.\n\n​\n\nTHURSDAY, 10/12\n\nPractice, Warren, 4:30–6:30\n\nLate: Cian\n\nOut: Bash, Matty, Zach\n\n​\n\nFRIDAY, 10/13\n\nScrimmage, Eliot, 4:15\n\nArrive at 3:45\n\nLate: —\n\nOut: Bash, Hayden, Mason, Theo, Zach\n\n​\n\nSATURDAY, 10/14\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Bash, Zach\n\n​\n\nSUNDAY, 10/15\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Bash,  Zach?\n\n​\n\nTUESDAY, 10/16\n\nPractice, Warren, 4:30–6:30\n\nL…Th/Sa/Su), September–November, at Warren Field (starting 9/5).\n\n \n\nWINTER 2024\n\nIndoor practices on Saturday or Sunday evenings, January–March, at Brookline HS Tappan Pavilion.\n\n \n\nSPRING 2024\n\n​Doubleheaders on Saturdays, April–June, in the 12U Division of the Select League.\n\n​\n\n\n\nPractices once or twice per week (TBD).\n\n\nPlayoffs in July, if we qualify.\n\n\nSchedule will not conflict with BYB Majors.\n\n\nPlaying time will depend on baseball skills, commitment, focus, work ethic, and attitude.​\n\n',
  ];
  unroll(`should be able to parse (#key) on the upcoming schedule`, 
      function(done, testArgs) {
        const result = parseSchedule(input[0]);
        expect(result.size !== 0);
        expect(result.has(testArgs['key']));
        const entry = result.get(testArgs['key']);
        expect(entry['dayOfWeek']).to.equal(testArgs['dayOfWeek']);
        expect(entry['dayOfMonth']).to.equal(testArgs['dayOfMonth']);
        expect(entry['location']).to.equal(testArgs['location']);
        expect(entry['timeBlock']).to.equal(testArgs['timeBlock']);
        expect(entry['parsed'][0].start.date()).to.eql(testArgs['parsedStartDate'].toDate());
        expect(entry['parsed'][0].end.date()).to.eql(testArgs['parsedEndDate'].toDate());
        done();
      },
      [
        ['key', 'dayOfWeek', 'dayOfMonth', 'location', 'timeBlock', 'parsedStartDate', 'parsedEndDate'],
        ['TUESDAY, 10/3', 'TUESDAY', '10/3', 'Practice, Warren', '4:45–6:45', moment('2023-10-03 16:45:00'), moment('2023-10-03 18:45:00')],
        ['THURSDAY, 10/5', 'THURSDAY', '10/5', 'Practice, Warren', '4:45–6:45', moment('2023-10-05 16:45:00'), moment('2023-10-05 18:45:00')],
        ['SATURDAY, 10/7', 'SATURDAY', '10/7', 'Practice, Warren', '3:00–5:30', moment('2023-10-07 15:00:00'), moment('2023-10-07 17:30:00')],
        ['SUNDAY, 10/8', 'SUNDAY', '10/8', 'Practice, Warren', '3:00–5:30', moment('2023-10-08 15:00:00'), moment('2023-10-08 17:30:00')],
      ],
  );

  it(`can parse a schedule entry that doesn't have a time block`, function() {
    const result = parseSchedule(input[2]);
    expect(result.size).to.equal(4);
    expect(result.get('SATURDAY, 10/7')['timeBlock']).to.equal(null);
    expect(result.get('SATURDAY, 10/7')['parsed']).to.equal(null);
    expect(result.get('SATURDAY, 10/7')['location']).to.equal('Practice is canceled');
  });

  it(`can parse a schedule entry that doesn't have a range for its time block`, function() {
    const result = parseSchedule(input[3]);
    expect(result.size).to.equal(5);
    expect(result.get('FRIDAY, 10/13')['timeBlock']).to.equal('4:15');
    expect(result.get('FRIDAY, 10/13')['parsed']).to.not.equal(null);
    expect(result.get('FRIDAY, 10/13')['parsed'][0].start.date()).to.eql(moment('2023-10-13 16:15:00').toDate());
    expect(result.get('FRIDAY, 10/13')['parsed'][0].end).to.equal(null);
    expect(result.get('FRIDAY, 10/13')['location']).to.equal('Scrimmage, Eliot');
  });

  it(`compares two schedules`, function() {
    const a = parseSchedule(input[0]);
    const b = parseSchedule(input[1]);
    const result = compareSchedules(a, b);
    expect(result['added'].size).to.equal(2); // added 10/10 and 10/12
    expect(result['deleted'].size).to.equal(1); // removed 10/3
    expect(result['modified'].size).to.equal(1); // modified 10/5 from 4:45pm start to 4:30pm start.
    expect(result['modified'].get('THURSDAY, 10/5')['timeBlock']).to.equal('4:30–6:30');
    expect(result['unchanged'].size).to.equal(2); // 10/7 and 10/8 remain unchanged
  });

  it(`can serialize and deserialize the schedule to disk`, function() {
    const a = parseSchedule(input[0]);
    serializeSchedule(a, path.join(__dirname, 'serializedTestSchedule.bson'));
    const b = deserializeSchedule(path.join(__dirname, 'serializedTestSchedule.bson'));
    const result = compareSchedules(a, b);
    expect(result['unchanged'].size).to.equal(4); // all entries unchanged
    expect(result['added'].size).to.equal(0);
    expect(result['deleted'].size).to.equal(0);
    expect(result['modified'].size).to.equal(0);
  });

  after(function() {
    if (fs.existsSync(path.join(__dirname, 'serializedTestSchedule.bson'))) {
      fs.unlinkSync(path.join(__dirname, 'serializedTestSchedule.bson'));
    }
  });
});
