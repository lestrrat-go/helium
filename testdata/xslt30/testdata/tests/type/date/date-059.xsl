<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs my"
xmlns:my="http://my.uri/">

<?spec fo#func-subtract-yearMonthDurations?>

  <!-- PURPOSE: XPath 2.0: test subtraction of two yearMonthDurations. -->
  
  <xsl:variable name="x">
  <a>P1Y</a>
  <a>P2Y</a>
  <a>P1Y6M</a>
  <a>P18M</a>
  <a>-P1Y</a>
  <a>-P1M</a> 
  </xsl:variable>

  <xsl:template match="/">
    <d>
        <xsl:for-each select="$x/a">
        <e>
            <xsl:variable name="dt" select="xs:yearMonthDuration(.)"/>
            <xsl:value-of select="for $z in ../a
                                return xs:yearMonthDuration($z) - $dt"
                                separator=", "/>
        </e>;
        </xsl:for-each>
    </d>                                       
  </xsl:template>
  


</xsl:stylesheet>