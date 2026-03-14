<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 >

<?spec serialization#xml-version?>
  <!-- PURPOSE: Error: serialize document containing XML 1.1 control characters as v 1.0-->

<xsl:output method="xml" version="1.0"/>

<xsl:template match="/">
  <xsl:copy-of select="//bell"/>
</xsl:template>

</xsl:stylesheet>
