<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:saxon="http://saxon.sf.net/">
  <xsl:accumulator name="minDate" initial-value="/*/block[1]/@date" as="xs:date">
    <xsl:accumulator-rule match="block" select="if(@date &lt; $value) then @date else $value"/>
  </xsl:accumulator>
  <xsl:mode use-accumulators="#all"/>
  <xsl:template match="/">
    <out><xsl:value-of select="accumulator-after('minDate')"/></out>
  </xsl:template>
</xsl:stylesheet>