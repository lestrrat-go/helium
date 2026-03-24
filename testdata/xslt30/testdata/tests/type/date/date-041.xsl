<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test less-than between times with different timezones. -->

  <xsl:template match="/">
    <out compare1="{xs:time('13:12:00Z')}" compare2="{xs:time('14:13:00+02:00')}">
      <xsl:value-of select="xs:time('13:12:00Z') &lt; xs:time('14:13:00+02:00')"/>
    </out>
  </xsl:template>

</xsl:stylesheet>