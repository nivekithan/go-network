CREATE TABLE plate_observation (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    plate_number TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    location INTEGER NOT NULL,
    road_id INTEGER NOT NULL,
    FOREIGN KEY (road_id) REFERENCES road(id)
);

CREATE TABLE road (
    id INTEGER PRIMARY KEY NOT NULL,
    speed_limit INTEGER NOT NULL
);

CREATE TABLE dispatcher (
    id INTEGER PRIMARY KEY NOT NULL,
    road_id INTEGER NOT NULL,
    dispatcher_id TEXT NOT NULL
);


CREATE TABLE ticket (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    plate_number TEXT NOT NULL,
    road_id INTEGER NOT NULL,
    mile_1 INTEGER NOT NULL,
    timestamp_1 INTEGER NOT NULL,
    mile_2 INTEGER NOT NULL,
    timestamp_2 INTEGER NOT NULL,
    speed INTEGER NOT NULL,


    -- range is inclusive
    day_start_range INTEGER NOT NULL,
    day_end_range INTEGER NOT NULL,

    -- it is getting used as boolean
    is_processed INTEGER NOT NULL,


    FOREIGN KEY (road_id) REFERENCES road(id)
);
