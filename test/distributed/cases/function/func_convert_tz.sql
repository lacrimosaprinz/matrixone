-- convert_tz timezone name
select convert_tz('2023-11-06 10:28:00','GMT', 'MET') as dtime;
select convert_tz('1999-06-06 10:28:00','MET','Europe/Moscow') as dtime;
select convert_tz('2020-05-09 10:28:00','Japan', 'Mexico/BajaNorte') as dtime;
select convert_tz('2023-08-06 10:28:00','MET','Europe/Moscow') as dtime;
select convert_tz('2007-09-11 02:00:00','America/Cambridge_Bay','GMT-0')as dtime;
select convert_tz('2000-10-06 10:28:00','GMT', 'UTC') as dtime;
select convert_tz('2003-12-06 10:28:00','CET','EST') as dtime;
select convert_tz('2023-12-31 10:28:00','+08:00', 'America/New_York') as dtime;
select convert_tz('2023-02-06 10:28:00','MET','Hongkong') as dtime;
select convert_tz('2023-03-11 02:00:00','Asia/Shanghai','+05:00')as dtime;
-- @bvt:issue#12593
select convert_tz('2007-03-11 02:00:00','US/Eastern','US/Central')as dtime;
-- @bvt:issue
select convert_tz('2023-11-05 05:00:00','US/Eastern','US/Central')as dtime;

-- convert_tz timezone
select convert_tz('2023-01-06 10:28:00','+08:00', '+10:00') as dtime;
select convert_tz('2023-02-06 10:28:00','+08:00', '+00:00') as dtime;
select convert_tz('2023-03-06 10:28:00','+08:00', '+05:00') as dtime;
select convert_tz('2023-04-26 10:28:00','+05:00', '+08:00') as dtime;
select convert_tz('2023-05-16 10:28:00','+00:00', '+08:00') as dtime;
select convert_tz('2023-06-01 10:28:00','+00:00', '+23:00') as dtime;
select convert_tz('2023-07-06 10:28:00','+06:00', '+12:00') as dtime;
select convert_tz('2023-08-30 10:28:00','+12:00', '+06:00') as dtime;
select convert_tz('2020-09-19 19:59:00','+00:00', '+05:30') as dtime;
select convert_tz('2020-10-19 19:59:00','-05:00', '+05:30') as dtime;
select convert_tz('2010-11-01 12:00:00','+00:00','-07:00') as dtime;
select convert_tz('2010-12-30','+00:00','-07:00') as dtime;

-- null
select convert_tz(NULL,'-05:00', '+05:30') as dtime;
select convert_tz('2023-11-06 10:28:00',NULL, '+08:00') as dtime;
select convert_tz('2023-11-06 10:28:00','+00:00', NULL) as dtime;

-- out of time range
select convert_tz('9999-12-31 23:59:59','+08:00', '+12:30') as dtime;
-- @bvt:issue#12513
select convert_tz('0001-01-01 00:00:01','+00:00', '-5:30') as dtime;
-- @bvt:issue
-- invalid parameter
select convert_tz('2023-11-06 10:28:00','+00:00', '11111') as dtime;
select convert_tz('2023-11-06 10:28:00','+00:aa', '+08:00') as dtime;
-- @bvt:issue#12513
select convert_tz('2023-11','+00:00', '+08:00') as dtime;
-- @bvt:issue
-- datetime,timestamp,date type
create table convert_table(c1 datetime,c2 date,c3 timestamp(3));
insert into convert_table values('2010-09-26','2022-01-02 10:02:00','2021-05-02 12:02:00.0923'),('2011-02-20 10:02:00','2020-01-02','2021-05-02'),('2019-03-16 11:12:00','2022-01-02 10:02:00','2021-05-02 12:02:00.0923');
select convert_tz(c1,'+00:00', '+08:00'),c1 from convert_table;
select convert_tz(c2,'+00:00', '+08:00'),c2 from convert_table;
select convert_tz(c3,'+00:00', '+08:00'),c3 from convert_table;

--date function
select convert_tz(str_to_date('2022-05-27 11:30:00','%Y-%m-%d %H:%i:%s'),'-05:00', '+05:30')as dtime;
