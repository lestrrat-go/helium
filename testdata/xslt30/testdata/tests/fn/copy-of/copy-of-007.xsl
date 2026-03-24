<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <!-- FileName: copy2118 -->
    <!-- Purpose: Use copy-of() to copy a document node; and check the result of deep-equal() -->


<xsl:template match="/">
  <out>
    <xsl:sequence select="deep-equal(., copy-of(.))"/>
  </out>
</xsl:template>

</xsl:stylesheet>