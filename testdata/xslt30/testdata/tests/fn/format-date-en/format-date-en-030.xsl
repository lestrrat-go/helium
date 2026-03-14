<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English AM/PM width-limited -->

  <xsl:param name="b" as="xs:time" select="xs:time('00:10:00')"/>

  <xsl:template name="main">
<out>;
<xsl:for-each select="0 to 23">
<xsl:variable name="t" select="$b + xs:dayTimeDuration('PT1H')*."/>
<z><xsl:value-of select="translate(format-time($t, '[h]~[m][P]', 'en', (), ()), '.- ', '')"/></z>;
</xsl:for-each>
</out>     
  </xsl:template>

</xsl:stylesheet>