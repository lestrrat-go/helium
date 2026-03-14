<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes"
xmlns:z="http://z.com/" exclude-result-prefixes="xs xdt z">

<?spec fo#func-sum?>
  <!-- PURPOSE: XSLT 2.0: test sum() over yearMonthDurations -->

    <xsl:param name="d1" as="xs:yearMonthDuration" select="xs:yearMonthDuration('P1Y3M')"/>
    <xsl:param name="d2" as="xs:yearMonthDuration" select="xs:yearMonthDuration('P15M')"/>
    <xsl:param name="d3" as="xs:yearMonthDuration" select="xs:yearMonthDuration('-P1M')"/>
    <xsl:param name="z" as="xs:yearMonthDuration" select="xs:yearMonthDuration('P0M')"/>

  <xsl:template match="/">
     <out>
      <a><xsl:value-of select="sum(($d1,$d2,$d3))"/></a>
      <a><xsl:value-of select="sum(($d1,$d2,$d3), $z)"/></a>
      <a><xsl:value-of select="sum((), $z)"/></a>   
      <a><xsl:value-of select="sum(($d1, $d2, $d3)[. gt $d1], ())"/></a>                  
     </out>     
  </xsl:template>

</xsl:stylesheet>
