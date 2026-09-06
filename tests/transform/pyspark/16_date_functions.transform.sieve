// Date functions: year, month, day
date_functions
    from s3
        bucket events
        region us-east-1
        prefix logs
        format delta

    extract
        json select message
            timestamp timestamp date(YYYY-MM-DD)

    select
        timestamp
        timestamp year date(YYYY) or year()
        timestamp month date(YYYY-MM) or month()
        timestamp day date(YYYY-MM-DD) or day()