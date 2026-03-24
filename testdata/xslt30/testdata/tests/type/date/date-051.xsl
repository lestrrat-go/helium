<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#func-dayTimeDuration-equal?>
  <!-- PURPOSE: XPath 2.0: test equality of durations. -->

  <xsl:template match="/">
    <d>
        <true x="{xs:duration('P12M')=xs:duration('P0Y12M')}"/>;
        <true x="{xs:duration('P12M')=xs:duration('P1Y')}"/>;
        <true x="{xs:duration('P1Y12M')=xs:duration('P24M')}"/>;
        <true x="{xs:duration('-P12M')=xs:duration('-P1Y')}"/>;
        <true x="{xs:duration('P1D')=xs:duration('PT24H')}"/>;
        <true x="{xs:duration('-P1DT12H')=xs:duration('-PT36H')}"/>;
    </d>                                       
  </xsl:template>

</xsl:stylesheet>