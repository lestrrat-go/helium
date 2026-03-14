<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test creation and display of durations. -->

  <xsl:template match="/">
    <d>
        <a><xsl:value-of select="xs:duration('P10Y5M17D')"/></a>;
        <a><xsl:value-of select="xs:duration('P10Y5M17DT10H5M3S')"/></a>;
        <a><xsl:value-of select="xs:duration('PT10.6S')"/></a>;
        <a><xsl:value-of select="xs:duration('P5DT3H4M')"/></a>;
        <a><xsl:value-of select="xs:duration('-P100D')"/></a>;
        <a><xsl:value-of select="xs:duration('PT0.5S')"/></a>; 
    </d>                                       
  </xsl:template>

</xsl:stylesheet>