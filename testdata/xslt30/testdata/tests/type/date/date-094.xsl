<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes"
    exclude-result-prefixes="#all">
  
    <!--
        For: xs:date()
        Tests different (in)valid values in corner cases for date time ranges, depending 
        on proper support for negative, zero or high (above 9999) date ranges.
        Also tests common date and date-time formats with invalid facet values.
     -->

    <xsl:param name="test-case" required="yes" static="yes"/>
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:value-of _select="{$test-case}" />
        </out>
    </xsl:template>

</xsl:stylesheet>
