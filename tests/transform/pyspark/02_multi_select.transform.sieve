// JSON select with multiple columns and types
multi_select
    from s3
        bucket analytics
        region us-west-2
        prefix events
        format delta

    extract
        json select message
            user_id user_id string
            event_type event_type string
            amount amount decimal(18,2)
            created_at created_at date(YYYY-MM-DD)
            count count bigint