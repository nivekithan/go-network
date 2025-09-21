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
