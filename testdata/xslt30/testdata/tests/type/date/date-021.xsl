<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-current-date?>
  <!-- PURPOSE: XPath 2.0: current-date() function. -->
  <!-- Also implicitly tests date to string conversion -->

  <xsl:template match="/">
    <out><xsl:value-of select="current-date() gt xs:date('2000-01-01')"/></out>
  </xsl:template>

</xsl:stylesheet>