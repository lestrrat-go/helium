<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#func-years-from-duration?>
  <?spec fo#func-months-from-duration?>
  <?spec fo#func-days-from-duration?>
  <?spec fo#func-hours-from-duration?>
  <?spec fo#func-minutes-from-duration?>
  <?spec fo#func-seconds-from-duration?>
  
  <!-- PURPOSE: XPath 2.0: test extraction of components of a duration. -->

  <xsl:template match="/">
    <d>
        <xsl:variable name="d" select="(xs:yearMonthDuration('P1Y2M'), xs:yearMonthDuration('-P1Y2M'))"/>
        <xsl:for-each select="$d">
        <e
            y="{years-from-duration(.)}"
            M="{months-from-duration(.)}"
         />;
        </xsl:for-each>                                               
        <xsl:variable name="d2" select="(xs:dayTimeDuration('P3DT4H5M6.7S'), xs:dayTimeDuration('-P3DT4H5M6.7S'))"/>
        <xsl:for-each select="$d2">
        <e
            d="{days-from-duration(.)}"
            h="{hours-from-duration(.)}"
            m="{minutes-from-duration(.)}"
            s="{seconds-from-duration(.)}" />;
        </xsl:for-each>                                                                       
    </d>                                       
  </xsl:template>

</xsl:stylesheet>