// Sink to S3 with partitionBy and overwrite mode
full_pipeline
    from s3
        bucket raw
        region us-east-1
        prefix events
        format delta

    extract
        json select message
            event_id event_id string
            event_date event_date date(YYYY-MM-DD)
            value value decimal(18,2)

    select
        event_id
        event_date
        event_date year date(YYYY) or year()
        event_date month date(YYYY-MM) or month()
        event_date day date(YYYY-MM-DD) or day()
        value

    to s3
        bucket processed
        region us-east-1
        prefix events_enriched
        format delta
        mode overwrite
        partitioned by year, month, day