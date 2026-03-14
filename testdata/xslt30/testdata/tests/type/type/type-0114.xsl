<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/02/xpath-datatypes"
exclude-result-prefixes="xs xdt">

  <!-- PURPOSE: Test numeric subtypes with large values. -->

  <xsl:template match="/">
  <out>
    <xsl:variable name="v1" select="xs:positiveInteger(123456789012345678901234567890)"/>
    <xsl:variable name="v2" select="xs:nonNegativeInteger('123456789012345678901234567890')"/>   
    <xsl:variable name="v3" select="xs:negativeInteger(-123456789012345678901234567890)"/>  
    <xsl:variable name="v4" select="xs:nonPositiveInteger('-123456789012345678901234567890')"/>  
    <xsl:variable name="v5" select="xs:unsignedLong(18446744073709551615)"/>
    <a><xsl:value-of select="$v1 instance of xs:positiveInteger"/></a>    
    <a><xsl:value-of select="$v1 instance of xs:negativeInteger"/></a>    
    <a><xsl:value-of select="$v2 instance of xs:unsignedLong"/></a>    
    <a><xsl:value-of select="$v2 instance of xs:nonNegativeInteger"/></a>
    <b><xsl:value-of select="$v3 = $v4"/></b>
    <b><xsl:value-of select="$v4 + $v5"/></b>
    <c><xsl:value-of select="18446744073709551616 castable as xs:unsignedLong"/></c>
    <c><xsl:value-of select="-123456789012345678901234567890 castable as xs:positiveInteger"/></c>    
    <c><xsl:value-of select="123456789012345678901234567890 castable as xs:negativeInteger"/></c>                                
  </out>     
  </xsl:template>                             


</xsl:stylesheet>