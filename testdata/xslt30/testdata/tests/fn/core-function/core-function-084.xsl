<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Purpose: Test of 'round' function with positive infinity
       as the argument (use expression that yields infinity). -->

<xsl:template match="doc">
 <out>
   <xsl:choose>
   <xsl:when test="system-property('xsl:version')='1.0'">
        <xsl:value-of select="round(2.3 div 0)"/>
   </xsl:when>
   <xsl:otherwise>
        <xsl:value-of select="round(2.3 div 0e0)"/>
   </xsl:otherwise>
   </xsl:choose>
 </out>
</xsl:template>
</xsl:stylesheet>