<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes"
xmlns:z="http://z.com/" exclude-result-prefixes="xs z xdt">

<?spec fo#func-divide-yearMonthDuration?>
  <!-- PURPOSE: division of yearMonthDurations -->


  <xsl:template match="/">
     <out>
       <x><xsl:value-of select="xs:yearMonthDuration('P1Y12M') div xs:yearMonthDuration('P1M')"/></x>
       <x><xsl:value-of select="xs:yearMonthDuration('-P10Y12M') div xs:yearMonthDuration('P1M')"/></x>
       <x><xsl:value-of select="xs:yearMonthDuration('P3M') div xs:yearMonthDuration('P100Y')"/></x>              
     </out>
  </xsl:template>

</xsl:stylesheet>