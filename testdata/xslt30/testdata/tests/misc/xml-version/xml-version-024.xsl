<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec serialization#xml-version?>
  <!-- PURPOSE: Path expression containing XML 1.1 name characters -->

<xsl:output method="xml" version="1.0"/>

<xsl:template match="/">
  <doc><xsl:value-of select="count(//r&#x133;k/@R&#x132;k)"/></doc>
</xsl:template>
 
</xsl:stylesheet>
