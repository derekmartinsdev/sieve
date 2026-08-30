// Derived source: from another section
source
    from s3
        bucket raw
        region us-east-1
        prefix base
        format delta

    extract
        json select message
            id id string
            value value decimal(18,2)

derived
    from source

    extract
        json select message
            id id string
            value value decimal(18,2)