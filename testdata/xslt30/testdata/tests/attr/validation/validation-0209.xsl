<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <xsl:template match="/">
    <out>
      <xsl:attribute name="wrong" type="xs:integer">fred</xsl:attribute>
    </out>
  </xsl:template>


</xsl:stylesheet>
