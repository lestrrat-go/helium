<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test equality between dates. -->

  <xsl:template match="/">
    <out><xsl:value-of select="xs:dateTime('1996-12-12T13:13:00Z') = 
                               xs:dateTime('1996-12-12T13:13:00+00:00')"/></out>
  </xsl:template>

</xsl:stylesheet>