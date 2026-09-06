// JSON select with nested paths (dotted source names)
nested_select
    from s3
        bucket orders
        region eu-west-1
        prefix raw
        format delta

    extract
        json select message
            customer.name customer_name string
            customer.address.city city string
            customer.address.zipcode zip string
            order.total total decimal(18,2)
            order.status status string