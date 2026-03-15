<xsl:stylesheet version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs">

<xsl:mode streamable="yes"/>

<xsl:template match="/">
 <out>
  <xsl:iterate select="doc/chapter/p">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:copy-of select="node()"/>
    </xsl:copy>
    <xsl:if test="position() ge 2">
      <xsl:break/>
    </xsl:if>
  </xsl:iterate>
 </out> 
</xsl:template>

</xsl:stylesheet>