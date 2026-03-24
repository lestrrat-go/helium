<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="3.0"
  xmlns:array="http://www.w3.org/2005/xpath-functions/array"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="xsl:initial-template">
    <xsl:variable as="xs:integer" name="m" select="[1]?1" />
    <xsl:variable as="array(xs:integer)" name="a" select="array:insert-before([2], 1, $m)" />
    <out a="{$a}"/>
  </xsl:template>
</xsl:stylesheet>