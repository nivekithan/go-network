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


-- name: ConflictingTickets :one
SELECT id FROM ticket WHERE
    plate_number = @plate_number AND
    (
    day_start_range BETWEEN @start_date AND @end_date OR
    day_end_rage BETWEEN @start_date AND @end_date
    ) LIMIT 1;

-- name: FindDispatcherForRoad :one
SELECT dispatcher_id FROM dispatcher WHERE road_id = @road_id LIMIT 1;


-- name: StoreTicket :exec
INSERT INTO ticket
    (plate_number, road_id, mile_1, timestamp_1, mile_2, timestamp_2, speed, day_start_range, day_end_range, is_processed)
VALUES (
    @plate_number, @road_id, @mile_1, @timestamp_1, @mile_2, @timestamp_2, @speed, @day_start_range, @day_end_range, @is_processed
);

-- name: GetUnProcessedTickets :many
SELECT * FROM ticket WHERE is_processed = 0;


-- name: MarkTicketAsProcessed :exec
UPDATE ticket SET is_processed = 1 WHERE id = @id;


-- name: AddDispatcherForRoad :exec
INSERT INTO dispatcher (road_id, dispatcher_id) VALUES (@road_id, @dispatcher_id);
