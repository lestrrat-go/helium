<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:output method="adaptive"/>
    
    <xsl:param name="json" as="xs:string">
        {
            "locations": [
                {
                    "id": "i1",
                    "types": [
                            {"name": "foo"},
                            {"name": "bar"}
                              ]
                },
                {
                    "id": "i2",
                    "types": [
                            {"name": "baz"}
                            ]
                },
                {
                    "id": "i3",
                    "types": [
                            {"name": "foo"},
                            {"name": "baz"}
                            ]
                }
            ]
        }
    </xsl:param>
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:sequence select="parse-json($json)?locations?*[?types?*[?name = 'foo']]"/>
    </xsl:template>
    
</xsl:stylesheet>