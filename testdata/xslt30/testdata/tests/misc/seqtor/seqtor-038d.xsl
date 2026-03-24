<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- 
        original purpose: to show that a sequence of two consecutive empty strings contains a space in-between and 
                 that adjacent text nodes are merged with no space in-between, but document nodes are not
    -->
  
    <!--
        MHK Note 2020-02-18. What the test does has very little resemblance to the stated
        purpose above. (The test does not produce two consecutive empty strings, nor
        does it manipulated document nodes.) I have changed the variable name from "non-empty"
        to "zero-length-string" for clarity. I have also changed the expected results.
        After atomization, the content sequence is ("", "|", "", "|", ""), and after
        inserting spaces between adjacent strings the result (with space represented as ~) is
        the string-join of ("", "~", "|", "~", "", "~", "|", "~", ""), which evaluates to
        " |  | ". The previous expected result was " | | ". -->
    
    <xsl:template match="/" name="xsl:initial-template">
        <xsl:variable name="zero-length-string" as="xs:string" select="''"/>        
        
        <bar>
            <xsl:namespace name="foo">
                <xsl:sequence select="$zero-length-string"/>
                <xsl:value-of select="'|'" />
                <xsl:sequence select="$zero-length-string"/>
                <xsl:value-of select="'|'" />
                <xsl:sequence select="$zero-length-string"/>
            </xsl:namespace>
        </bar>
    </xsl:template>
    
</xsl:stylesheet>