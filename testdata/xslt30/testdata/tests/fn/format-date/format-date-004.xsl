<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-time: 12-hour clock -->

  <xsl:param name="t" as="xs:time" select="xs:time('09:15:06.456')"/>  

  <xsl:template name="main">
<out>;
<xsl:for-each select="1 to 24">
<t><xsl:value-of select="format-time($t + xs:dayTimeDuration('PT1H')*., '[h].[m]')"/></t>;
</xsl:for-each>
</out>     
  </xsl:template>

</xsl:stylesheet>