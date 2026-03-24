<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on arithmetic expressions -->

  <xsl:template name="main" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
    <out><xsl:value-of select="(1 + 1) instance of xs:double"/></out>
  </xsl:template>


</xsl:stylesheet>
