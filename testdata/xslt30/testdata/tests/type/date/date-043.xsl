<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test less-than between times with date wraparound. -->
  <!-- (The answer, surprisingly, is "false", because of conversion to GMT) -->

  <xsl:template match="/">
    <false><xsl:value-of select="xs:time('18:00:00-05:00') &lt; 
                               xs:time('20:00:00-05:00')"/></false>
  </xsl:template>

</xsl:stylesheet>