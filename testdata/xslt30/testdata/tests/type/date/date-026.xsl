<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: time constructor with no timezone. -->
  <!-- Also implicitly tests date to string conversion -->

  <xsl:template match="/">
    <out><xsl:value-of select="xs:time('12:00:00.006')"/></out>
  </xsl:template>

</xsl:stylesheet>