<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: dateTime constructor with milliseconds and timezone. -->
  <!-- Also implicitly tests date to string conversion -->

  <xsl:template match="/">
    <out><xsl:value-of select="string(xs:dateTime('2000-01-01T12:00:00.1+05:00'))"/></out>
  </xsl:template>

</xsl:stylesheet>