<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-current-time?>
  <!-- PURPOSE: XPath 2.0: current-time() function. -->
  <!-- Also tests time to string conversion -->

  <xsl:template match="/">
    <out><xsl:value-of select="matches(string(current-time()), '[012][0-9]:[0-5][0-9]:[0-5][0-9](.[0-9]*)?(Z|([+-][01][0-9]:[0-5][0-9]))')"/></out>
  </xsl:template>

</xsl:stylesheet>