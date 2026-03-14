<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test sorting of durations. -->
  
  <xsl:variable name="x">
  <a>P1Y</a>
  <a>P12M</a>
  <a>P365D</a>
  <a>P366D</a>
  <a>P1M</a>
  <a>P28D</a>
  <a>P29D</a>
  <a>P30D</a>
  <a>P31D</a>
  <a>P28DT12H</a>
  <a>PT3600S</a>
  <a>P28DT11H65M</a>
  </xsl:variable>

  <xsl:template match="/">
    <d>
        <xsl:for-each select="$x/a">
        <xsl:sort select="xs:duration(.)" />
            <a><xsl:value-of select="."/></a>;
        </xsl:for-each>
    </d>                                       
  </xsl:template>

</xsl:stylesheet>