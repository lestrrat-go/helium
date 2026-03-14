<?xml version="1.0"?>
<!-- test xsl:evaluate calling an extension function -->
<!-- formerly test saxon037 -->

<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:saxon="http://saxon.sf.net/" 
   xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
   xmlns:xs="http://www.w3.org/2001/XMLSchema" 
   exclude-result-prefixes="saxon xs math">

   <xsl:param name="exp" select="
     if (function-available('saxon:system-id', 0))
     then 'saxon:system-id()' 
     else 'string(math:pow(12, 3))'"/>
    
   <xsl:variable name="id" as="xs:string">
      <xsl:evaluate xpath="$exp" context-item="."/>
   </xsl:variable>
    
   <xsl:template match="/" name="xsl:initial-template">

      <out>      
        <xsl:choose use-when="function-available('saxon:system-id', 0)">
          <xsl:when test="$id = saxon:system-id()">OK</xsl:when>
          <xsl:otherwise>TEST FAILED: <xsl:value-of select="saxon:system-id()"/></xsl:otherwise>
        </xsl:choose>
        <xsl:choose use-when="not(function-available('saxon:system-id', 0))">
          <xsl:when test="'1728' = $id">OK</xsl:when>
          <xsl:otherwise>TEST FAILED"/></xsl:otherwise>
        </xsl:choose>
      </out>
   </xsl:template>

</xsl:stylesheet>

