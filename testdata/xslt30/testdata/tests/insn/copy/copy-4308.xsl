<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 
  
<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template name="xsl:initial-template">
  <out>
    <xsl:call-template name="temp"/>
    <xsl:copy-of select="$var"/>
  </out>
</xsl:template>

<xsl:template name="temp">
  <xsl:param name="b" select="false()"/>
  <xsl:if test="$b">
    <xsl:copy>
      <in/>
    </xsl:copy>
  </xsl:if>
</xsl:template>
  
<xsl:variable name="var">
  <xsl:call-template name="temp">
    <xsl:with-param name="b" select="true()"/>
  </xsl:call-template>
</xsl:variable>
  
</xsl:stylesheet>