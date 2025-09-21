-- name: InsertRoad :exec
INSERT INTO road (id, speed_limit) VALUES (@id, @speed_limit);

-- name: GetRoad :one
SELECT * FROM road WHERE id = @id;

-- name: InsertPlateObservation :one
INSERT INTO plate_observation
    (plate_number, road_id, timestamp, location) VALUES
    (@plate_number, @road_id, @timestamp, @location)
RETURNING id;

-- name: GetPreviousObservation :one
SELECT * FROM plate_observation WHERE
    plate_number = @plate_number AND
    road_id = @road_id AND
    timestamp < @timestamp
ORDER BY timestamp DESC LIMIT 1;

-- name: GetNextObservation :one
SELECT * FROM plate_observation WHERE
    plate_number = @plate_number AND
    road_id = @road_id AND
    timestamp > @timestamp
ORDER BY timestamp ASC LIMIT 1;

-- name: GetObservationById :one
SELECT * FROM plate_observation WHERE id = @id;
