<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:j="http://www.w3.org/2005/xpath-functions">

    <!-- Tests for handling of duplicate keys in json-to-xml --> 

    <xsl:param name="duplicates" required="yes"/>

    <xsl:param name="in" as="xs:string" expand-text="no">
       {"one": {"a":2, "b":3, "c":4, "a":5},
        "two": {"a":2, "b":3, "c":4, "a":5},
        "one": {"a":3, "b":4, "c":5, "a":6},
        "two": {"a":3, "b":4, "c":5, "a":6}
       }
        
    </xsl:param>    

    <xsl:template name="xsl:initial-template">
        <xsl:sequence select="json-to-xml($in, map{'duplicates' : $duplicates })"/>
    </xsl:template>


</xsl:stylesheet>
