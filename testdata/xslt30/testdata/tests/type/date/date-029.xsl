<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: date constructor with positive and negative timezones. -->
  <!-- Also implicitly tests date to string conversion -->

  <xsl:template match="/">
    <out>
     <a><xsl:value-of select="xs:date('2000-01-01+02:30')"/></a>
     <a><xsl:value-of select="xs:date('2000-04-30-02:30')"/></a>
    </out>     
  </xsl:template>

</xsl:stylesheet>