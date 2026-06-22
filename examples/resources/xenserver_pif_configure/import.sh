terraform import xenserver_pif_configure.pif_update 00000000-0000-0000-0000-000000000000

# when use 'for_each' in resource
terraform  import  xenserver_pif_configure.pif_update[\"{each.key}\"] 00000000-0000-0000-0000-000000000000