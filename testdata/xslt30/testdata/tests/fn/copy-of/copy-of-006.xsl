<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <!-- FileName: copy2117 -->
    <!-- Purpose: Use copy-of() to copy a text node; and check the result of generate-id() -->


<xsl:template match="/">
  <xsl:variable name="att" as="attribute()">
    <xsl:attribute name="a" select="'2010-01-01'" type="xs:date"/>
  </xsl:variable>  
  <out>
    <xsl:sequence select="generate-id(copy-of((//text())[1])) = generate-id((//text())[1])"/>
  </out>
</xsl:template>

</xsl:stylesheet>